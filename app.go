package main

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"encoding/json"

	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/native-ingester/native"
	"github.com/Financial-Times/native-ingester/queue"
	"github.com/Financial-Times/native-ingester/resources"
	"github.com/Financial-Times/service-status-go/httphandlers"
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

func main() {
	app := cli.App("Native Ingester", "A service to ingest native content of any type and persist it in the native store, then, if required forwards the message to a new message queue")

	//Read queue configuration
	readQueueAddresses := app.Strings(cli.StringsOpt{
		Name:   "read-queue-addresses",
		Value:  nil,
		Desc:   "Addresses to connect to the consumer queue (URLs).",
		EnvVar: "Q_READ_ADDR",
	})
	readQueueGroup := app.String(cli.StringOpt{
		Name:   "read-queue-group",
		Value:  "",
		Desc:   "Group used to read the messages from the queue.",
		EnvVar: "Q_GROUP",
	})
	readQueueTopic := app.String(cli.StringOpt{
		Name:   "read-queue-topic",
		Value:  "",
		Desc:   "The topic to read the meassages from.",
		EnvVar: "Q_READ_TOPIC",
	})
	readQueueHostHeader := app.String(cli.StringOpt{
		Name:   "read-queue-host-header",
		Value:  "kafka",
		Desc:   "The host header for the queue to read the meassages from.",
		EnvVar: "Q_READ_QUEUE_HOST_HEADER",
	})
	readQueueConcurrentProcessing := app.Bool(cli.BoolOpt{
		Name:   "read-queue-concurrent-processing",
		Value:  false,
		Desc:   "Whether the consumer uses concurrent processing for the messages",
		EnvVar: "Q_READ_CONCURRENT_PROCESSING",
	})

	// Native writer configuration
	nativeWriterAddress := app.String(cli.StringOpt{
		Name:   "native-writer-address",
		Value:  "",
		Desc:   "Address of service that writes persistently the native content",
		EnvVar: "NATIVE_RW_ADDRESS",
	})
	nativeWriterCollectionsByOrigins := app.String(cli.StringOpt{
		Name:   "native-writer-collections-by-origins",
		Value:  "[]",
		Desc:   "Map in a JSON-like format. originId referring the collection that the content has to be persisted in. e.g. [{\"http://cmdb.ft.com/systems/methode-web-pub\":\"methode\"}]",
		EnvVar: "NATIVE_RW_COLLECTIONS_BY_ORIGINS",
	})
	nativeWriterHostHeader := app.String(cli.StringOpt{
		Name:   "native-writer-host-header",
		Value:  "nativerw",
		Desc:   "coco-specific header needed to reach the destination address",
		EnvVar: "NATIVE_RW_HOST_HEADER",
	})
	contentUUIDfields := app.Strings(cli.StringsOpt{
		Name:   "content-uuid-fields",
		Value:  []string{},
		Desc:   "List of JSONPaths that point to UUIDs in native content bodies. e.g. uuid,post.uuid,data.uuidv3",
		EnvVar: "NATIVE_CONTENT_UUID_FIELDS",
	})

	// Write Queue configuration
	writeQueueAddress := app.String(cli.StringOpt{
		Name:   "write-queue-address",
		Value:  "",
		Desc:   "Address to connect to the producer queue (URL).",
		EnvVar: "Q_WRITE_ADDR",
	})
	writeQueueTopic := app.String(cli.StringOpt{
		Name:   "write-topic",
		Value:  "",
		Desc:   "The topic to write the meassages to.",
		EnvVar: "Q_WRITE_TOPIC",
	})
	writeQueueHostHeader := app.String(cli.StringOpt{
		Name:   "write-queue-host-header",
		Value:  "kafka",
		Desc:   "The host header for the queue to write the meassages to.",
		EnvVar: "Q_WRITE_QUEUE_HOST_HEADER",
	})

	app.Action = func() {

		srcConf := consumer.QueueConfig{
			Addrs:                *readQueueAddresses,
			Group:                *readQueueGroup,
			Topic:                *readQueueTopic,
			Queue:                *readQueueHostHeader,
			ConcurrentProcessing: *readQueueConcurrentProcessing,
		}

		var collectionsByOriginIds map[string]string
		if err := json.Unmarshal([]byte(*nativeWriterCollectionsByOrigins), &collectionsByOriginIds); err != nil {
			log.WithError(err).Error("Couldn't parse JSON for originId to collection map")
		}

		bodyParser := native.NewContentBodyParser(*contentUUIDfields)
		writer := native.NewWriter(*nativeWriterAddress, collectionsByOriginIds, *nativeWriterHostHeader, bodyParser)

		mh := queue.NewMessageHandler(writer)

		var messageProducer producer.MessageProducer
		if *writeQueueAddress != "" {
			producerConfig := producer.MessageProducerConfig{
				Addr:  *writeQueueAddress,
				Topic: *writeQueueTopic,
				Queue: *writeQueueHostHeader,
			}
			messageProducer = producer.NewMessageProducer(producerConfig)
			mh.ForwardTo(messageProducer)
		}

		messageConsumer := consumer.NewConsumer(srcConf, mh.HandleMessage, http.Client{})
		log.Infof("[Startup] Consumer: %# v", messageConsumer)
		log.Infof("[Startup] Using source configuration: %# v", srcConf)
		log.Infof("[Startup] Using native writer configuration: %# v", writer)
		log.Infof("[Startup] Using native writer configuration: %# v", *contentUUIDfields)

		go enableHealthCheck(messageConsumer, writer, messageProducer)
		startMessageConsumption(messageConsumer)
	}

	err := app.Run(os.Args)
	if err != nil {
		println(err)
	}
}

func enableHealthCheck(c consumer.MessageConsumer, nw native.Writer, p producer.MessageProducer) {
	hc := resources.NewHealthCheck(c, nw, p)

	r := mux.NewRouter()
	r.HandleFunc("/__health", hc.Handler())
	r.HandleFunc(httphandlers.GTGPath, hc.GTG).Methods("GET")
	r.HandleFunc(httphandlers.BuildInfoPath, httphandlers.BuildInfoHandler).Methods("GET")
	r.HandleFunc(httphandlers.PingPath, httphandlers.PingHandler).Methods("GET")

	http.Handle("/", r)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.WithError(err).Panic("Couldn't set up HTTP listener")
	}
}

func startMessageConsumption(messageConsumer consumer.MessageConsumer) {
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
