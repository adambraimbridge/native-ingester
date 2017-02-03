package main

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"encoding/json"

	"io/ioutil"

	"io"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/golang/go/src/pkg/bytes"
	"github.com/golang/go/src/pkg/strings"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"

	"github.com/jmoiron/jsonq"

	log "github.com/Sirupsen/logrus"
)

func init() {
	f := &log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	}

	log.SetFormatter(f)
}

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
	sourceUUIDFields := app.Strings(cli.StringsOpt{
		Name:   "source-uuid-fields",
		Value:  []string{},
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
		uuidFields = *sourceUUIDFields
		srcConf := consumer.QueueConfig{
			Addrs:                *sourceAddresses,
			Group:                *sourceGroup,
			Topic:                *sourceTopic,
			Queue:                *sourceQueue,
			ConcurrentProcessing: *sourceConcurrentProcessing,
		}

		var collectionsByOriginIds map[string]string
		if err := json.Unmarshal([]byte(*destinationCollectionsByOrigins), &collectionsByOriginIds); err != nil {
			log.WithError(err).Error("Couldn't parse JSON for originId to collection map")
		}
		nativeWriterConfig := NativeWriterConfig{
			Address:                *destinationAddress,
			CollectionsByOriginIds: collectionsByOriginIds,
			Header:                 *destinationHeader,
		}

		writerConfig = nativeWriterConfig
		log.Infof("[Startup] Using source configuration: %# v", srcConf)
		log.Infof("[Startup] Using native writer configuration: %# v", nativeWriterConfig)

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
		log.WithError(err).Panic("Couldn't set up HTTP listener")
	}
}

func readMessages(config consumer.QueueConfig) {
	messageConsumer := consumer.NewConsumer(config, handleMessage, http.Client{})
	log.Infof("[Startup] Consumer: %# v", messageConsumer)

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
		log.WithField("transaction_id", tid).WithField("Origin-System-Id", msg.Headers["Origin-System-Id"]).Info("Skipping content because of not whitelisted Origin-System-Id")
		return
	}
	contents := make(map[string]interface{})
	if err := json.Unmarshal([]byte(msg.Body), &contents); err != nil {
		log.WithError(err).WithField("transaction_id", tid).Error("Error unmarshalling message. Ignoring message.")
		return
	}

	log.Infof("uuidFields: %v", uuidFields)
	uuid := extractUUID(contents, uuidFields)
	if uuid == "" {
		log.WithField("transaction_id", tid).Error("Error extracting uuid. Ignoring message.")
		return
	}
	log.WithField("transaction_id", tid).WithField("uuid", uuid).Info("Start processing native publish event")

	requestURL := writerConfig.Address + "/" + coll + "/" + uuid
	log.WithField("transaction_id", tid).WithField("requestURL", requestURL).Info("Built request URL for native writer")

	contents["lastModified"] = time.Now().Round(time.Millisecond)

	bodyWithTimestamp, err := json.Marshal(contents)

	if err != nil {
		log.WithError(err).WithField("transaction_id", tid).Error("Error marshalling message")
		return
	}

	request, err := http.NewRequest("PUT", requestURL, bytes.NewBuffer(bodyWithTimestamp))
	if err != nil {
		log.WithError(err).WithField("transaction_id", tid).WithField("requestURL", requestURL).Error("Error calling native writer. Ignoring message.")
		return
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-Id", tid)
	if len(strings.TrimSpace(writerConfig.Header)) > 0 {
		request.Host = writerConfig.Header
	}

	response, err := client.Do(request)

	if err != nil {
		log.WithError(err).WithField("transaction_id", tid).WithField("requestURL", requestURL).Error("Error calling native writer. Ignoring message.")
		return
	}
	defer properClose(response)

	if response.StatusCode != http.StatusOK {
		log.WithField("transaction_id", tid).WithField("responseStatusCode", response.StatusCode).Error("Native writer returned non-200 code")
		return
	}

	log.WithField("transaction_id", tid).WithField("uuid", uuid).Info("Successfully finished processing native publish event")
}

func extractUUID(contents map[string]interface{}, uuidFields []string) string {
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
		log.WithError(err).Warn("Couldn't read response body")
	}
	err = resp.Body.Close()
	if err != nil {
		log.WithError(err).Warn("Couldn't close response body")
	}
}
