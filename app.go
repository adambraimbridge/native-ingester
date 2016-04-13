package main

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"encoding/json"

	"io/ioutil"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/golang/go/src/pkg/bytes"
	"github.com/golang/go/src/pkg/strings"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/kr/pretty"
)

// NativeWriterConfig Holds the configuration for the Native Writer
type NativeWriterConfig struct {
	Address                string
	CollectionsByOriginIds []CollectionByOriginId
	Header                 string
}

type CollectionByOriginId struct {
	OriginId   string `json:"originId"`
	Collection string `json:"collection"`
}

var uuidField string
var client = http.Client{}
var writerConfig NativeWriterConfig

func main() {
	app := cli.App("Native Ingester", "A service to ingest native content of any type and persist it in the native store")
	sourceAddresses := app.Strings(cli.StringsOpt{
		Name:   "source-addresses",
		Value:  []string{},
		Desc:   "Addresses used by the queue consumer to connect to the queue",
		EnvVar: "SRC_ADDR",
	})
	sourceGroup := app.String(cli.StringOpt{
		Name:   "source-group",
		Value:  "",
		Desc:   "Group used to read the messages from the queue",
		EnvVar: "SRC_GROUP",
	})
	sourceTopic := app.String(cli.StringOpt{
		Name:   "source-topic",
		Value:  "",
		Desc:   "The topic to read the meassages from",
		EnvVar: "SRC_TOPIC",
	})
	sourceQueue := app.String(cli.StringOpt{
		Name:   "source-queue",
		Value:  "",
		Desc:   "The queue to read the messages from",
		EnvVar: "SRC_QUEUE",
	})
	sourceUUIDField := app.String(cli.StringOpt{
		Name:   "source-uuid-field",
		Value:  "uuid",
		Desc:   "Field in the message containing the UUID",
		EnvVar: "SRC_UUID_FIELD",
	})
	sourceConcurrentProcessing := app.Bool(cli.BoolOpt{
		Name:   "source-concurrent-processing",
		Value:  false,
		Desc:   "Whether the consumer uses concurrent processing for the messages",
		EnvVar: "SRC_CONCURRENT_PROCESSING",
	})
	destinationAddress := app.String(cli.StringOpt{
		Name:   "destination-address",
		Value:  "",
		Desc:   "Where to persist the native content",
		EnvVar: "DEST_ADDRESS",
	})
	destinationCollectionsByOrigins := app.String(cli.StringOpt{
		Name:   "destination-collections-by-origins",
		Value:  "[]",
		Desc:   "JSON. originId that a content has. collection to persist the native content in. e.g. [{\"originId\":\"http://cmdb.ft.com/systems/methode-web-pub\",\"collection\":\"methode\"}]",
		EnvVar: "DEST_COLLECTIONS_BY_ORIGINS",
	})
	destinationHeader := app.String(cli.StringOpt{
		Name:   "destination-header",
		Value:  "nativerw",
		Desc:   "coco-specific header needed to reach the destination address",
		EnvVar: "DEST_HEADER",
	})

	app.Action = func() {
		uuidField = *sourceUUIDField
		srcConf := consumer.QueueConfig{
			Addrs:                *sourceAddresses,
			Group:                *sourceGroup,
			Topic:                *sourceTopic,
			Queue:                *sourceQueue,
			ConcurrentProcessing: *sourceConcurrentProcessing,
		}
		var collectionsByOriginIds []CollectionByOriginId
		if err := json.Unmarshal([]byte(*destinationCollectionsByOrigins), &collectionsByOriginIds); err != nil {
			errorLogger.Panicf("Couldn't parse JSON for originId to collection map: %v\n", err)
		}
		nativeWriterConfig := NativeWriterConfig{
			Address:                *destinationAddress,
			CollectionsByOriginIds: collectionsByOriginIds,
			Header:                 *destinationHeader,
		}
		writerConfig = nativeWriterConfig
		initLogs(os.Stdout, os.Stdout, os.Stderr)
		infoLogger.Printf("[Startup] Using source configuration: %# v", pretty.Formatter(srcConf))
		infoLogger.Printf("[Startup] Using native writer configuration: %# v", pretty.Formatter(nativeWriterConfig))

		go enableHealthChecks(srcConf, nativeWriterConfig)
		readMessages(srcConf)
	}

	app.Run(os.Args)
}

func enableHealthChecks(srcConf consumer.QueueConfig, nativeWriteConfig NativeWriterConfig) {
	healthCheck := &Healthcheck{
		client:           http.Client{},
		srcConf:          srcConf,
		nativeWriterConf: nativeWriteConfig}
	router := mux.NewRouter()
	router.HandleFunc("/__health", healthCheck.checkHealth())
	router.HandleFunc("/__gtg", healthCheck.gtg)
	http.Handle("/", router)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		errorLogger.Panicf("Couldn't set up HTTP listener: %v\n", err)
	}
}

func readMessages(config consumer.QueueConfig) {
	messageConsumer := consumer.NewConsumer(config, handleMessage, http.Client{})
	infoLogger.Printf("[Startup] Consumer: %# v", pretty.Formatter(messageConsumer))

	var consumerWaitGroup sync.WaitGroup
	consumerWaitGroup.Add(1)

	go func() {
		messageConsumer.Start()
		consumerWaitGroup.Done()
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	messageConsumer.Stop()
	consumerWaitGroup.Wait()
}

func handleMessage(msg consumer.Message) {
	tid := msg.Headers["X-Request-Id"]
	coll := findCollection(msg)
	if coll == "" {
		infoLogger.Printf("[%s] Skipping content because of not whitelisted Origin-System-Id: %s", tid, msg.Headers["Origin-System-Id"])
	}
	contents := make(map[string]interface{})

	if err := json.Unmarshal([]byte(msg.Body), &contents); err != nil {
		errorLogger.Printf("[%s] Error unmarshalling message: [%v]", tid, err.Error())
		return
	}

	uuid := contents[uuidField]
	infoLogger.Printf("[%s] Start  processing native publish event for uuid [%s]", tid, uuid)

	uuidString, ok := uuid.(string)

	if !ok {
		errorLogger.Printf("[%s] Error transforming uuid [%v] to string.", tid, uuid)
		return
	}

	requestURL := writerConfig.Address + "/" + coll + "/" + uuidString
	infoLogger.Printf("[%s] Request URL: [%s]", tid, requestURL)

	request, err := http.NewRequest("PUT", requestURL, bytes.NewBuffer([]byte(msg.Body)))
	if err != nil {
		errorLogger.Printf("[%s] Error caling writer at [%s]: [%v]", tid, requestURL, err.Error())
		return
	}

	request.Header.Set("Content-Type", "application/json")
	if len(strings.TrimSpace(writerConfig.Header)) > 0 {
		request.Host = writerConfig.Header
	}
	infoLogger.Printf("doing request %v %v %v %v", request.URL, request.Header["Content-Type"], request.Host, request.Method)

	response, err := client.Do(request)

	if err != nil {
		errorLogger.Printf("[%s] Error caling writer at [%s]: [%v]", tid, requestURL, err.Error())
		return
	}
	defer response.Body.Close()

	ioutil.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		errorLogger.Printf("[%s] Caller returned non-200 code: [%v]", tid, response.StatusCode)
		return
	}
	infoLogger.Printf("[%s] Finish processing native publish event for uuid [%s]", tid, uuid)
}

func findCollection(msg consumer.Message) string {
	for _, e := range writerConfig.CollectionsByOriginIds {
		if e.OriginId == msg.Headers["Origin-System-Id"] {
			return e.Collection
		}
	}
	return ""
}
