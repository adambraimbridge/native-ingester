package main

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"encoding/json"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"

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

		mh := newMessageHandler(*sourceUUIDFields, nativeWriterConfig)
		messageConsumer := consumer.NewConsumer(srcConf, mh.handleMessage, http.Client{})
		log.Infof("[Startup] Consumer: %# v", messageConsumer)
		log.Infof("[Startup] Using source configuration: %# v", srcConf)
		log.Infof("[Startup] Using native writer configuration: %# v", nativeWriterConfig)
		log.Infof("[Startup] Using native writer configuration: %# v", *sourceUUIDFields)

		go enableHealthChecks(srcConf, nativeWriterConfig)
		startMessageConsumption(messageConsumer)
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

func startMessageConsumption(messageConsumer consumer.Consumer) {
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
