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
	"io"

	"github.com/jmoiron/jsonq"
)

// NativeWriterConfig Holds the configuration for the Native Writer
type NativeWriterConfig struct {
	Address                string
	CollectionsByOriginIds map[string]string
	Header                 string
}

var uuidFields []string
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
	sourceUUIDField := app.Strings(cli.StringsOpt{
		Name:   "source-uuid-fields",
		Value:  []string {},
		Desc:   "List of JSONPaths to try for extracting the uuid from native message. e.g. uuid,post.uuid,data.uuidv3",
		EnvVar: "SRC_UUID_FIELDS",
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
		Desc:   "Map in a JSON-like format. originId referring the collection that the content has to be persisted in. e.g. [{\"http://cmdb.ft.com/systems/methode-web-pub\":\"methode\"}]",
		EnvVar: "DEST_COLLECTIONS_BY_ORIGINS",
	})
	destinationHeader := app.String(cli.StringOpt{
		Name:   "destination-header",
		Value:  "nativerw",
		Desc:   "coco-specific header needed to reach the destination address",
		EnvVar: "DEST_HEADER",
	})

	app.Action = func() {
		uuidFields = *sourceUUIDField
		srcConf := consumer.QueueConfig{
			Addrs:                *sourceAddresses,
			Group:                *sourceGroup,
			Topic:                *sourceTopic,
			Queue:                *sourceQueue,
			ConcurrentProcessing: *sourceConcurrentProcessing,
		}
		initLogs(os.Stdout, os.Stdout, os.Stderr)
		var collectionsByOriginIds map[string]string
		if err := json.Unmarshal([]byte(*destinationCollectionsByOrigins), &collectionsByOriginIds); err != nil {
			errorLogger.Panicf("Couldn't parse JSON for originId to collection map: %v\n", err)
		}
		nativeWriterConfig := NativeWriterConfig{
			Address:                *destinationAddress,
			CollectionsByOriginIds: collectionsByOriginIds,
			Header:                 *destinationHeader,
		}
		writerConfig = nativeWriterConfig
		infoLogger.Printf("[Startup] Using source configuration: %# v", pretty.Formatter(srcConf))
		infoLogger.Printf("[Startup] Using native writer configuration: %# v", pretty.Formatter(nativeWriterConfig))

		go enableHealthChecks(srcConf, nativeWriterConfig)
		readMessages(srcConf)
	}

	err := app.Run(os.Args)
	if err != nil {
		println(err)
	}
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
	coll := writerConfig.CollectionsByOriginIds[strings.TrimSpace(msg.Headers["Origin-System-Id"])]
	if coll == "" {
		infoLogger.Printf("[%s] Skipping content because of not whitelisted Origin-System-Id: %s", tid, msg.Headers["Origin-System-Id"])
		return
	}
	contents := make(map[string]interface{})
	if err := json.Unmarshal([]byte(msg.Body), &contents); err != nil {
		errorLogger.Printf("[%s] Error unmarshalling message. Ignoring message. : [%v]", tid, err.Error())
		return
	}

	uuid := extractUuid(contents, uuidFields)
	if uuid == "" {
		errorLogger.Printf("[%s] Error transforming uuid [%v] to string. Ignoring message.", tid, uuid)
		return
	}
	infoLogger.Printf("[%s] Start processing native publish event for uuid [%s]", tid, uuid)

	requestURL := writerConfig.Address + "/" + coll + "/" + uuid
	infoLogger.Printf("[%s] Request URL: [%s]", tid, requestURL)
	request, err := http.NewRequest("PUT", requestURL, bytes.NewBuffer([]byte(msg.Body)))
	if err != nil {
		errorLogger.Printf("[%s] Error caling writer at [%s] Ignoring message.: [%v]", tid, requestURL, err.Error())
		return
	}
	request.Header.Set("Content-Type", "application/json")
	if len(strings.TrimSpace(writerConfig.Header)) > 0 {
		request.Host = writerConfig.Header
	}
	response, err := client.Do(request)
	if err != nil {
		errorLogger.Printf("[%s] Error caling writer at [%s] Ignoring message.: [%v]", tid, requestURL, err.Error())
		return
	}
	defer properClose(response)

	if response.StatusCode != http.StatusOK {
		errorLogger.Printf("[%s] Caller returned non-200 code: [%v] . Ignoring message.", tid, response.StatusCode)
		return
	}
	infoLogger.Printf("[%s] Successfully finished processing native publish event for uuid [%s]", tid, uuid)
}

func extractUuid(contents map[string]interface{}, uuidFields []string) string {
	jq := jsonq.NewQuery(contents)
	for _, uuidField := range uuidFields {
		uuid, err := jq.String(strings.Split(uuidField, ".")...)
		if err == nil && uuid != "" {
			return uuid
		}
	}
	return ""
}

func properClose(resp *http.Response) {
	_, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		warnLogger.Printf("Couldn't read response body: %v.", err)
	}
	err = resp.Body.Close()
	if err != nil {
		warnLogger.Printf("Couldn't close response body: %v.", err)
	}
}
