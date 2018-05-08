package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/kafka-client-go/kafka"
	"github.com/Financial-Times/native-ingester/native"
	"github.com/Financial-Times/native-ingester/queue"
	"github.com/Financial-Times/native-ingester/resources"
	"github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
)

func main() {
	app := cli.App("native-ingester", "A service to ingest native content of any type and persist it in the native store, then, if required forwards the message to a new message queue")

	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "Port to listen on",
		EnvVar: "PORT",
	})

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
		EnvVar: "Q_READ_GROUP",
	})
	readQueueTopic := app.String(cli.StringOpt{
		Name:   "read-queue-topic",
		Value:  "",
		Desc:   "The topic to read the messages from.",
		EnvVar: "Q_READ_TOPIC",
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
		Value:  "",
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
		Desc:   "The topic to write the messages to.",
		EnvVar: "Q_WRITE_TOPIC",
	})
	contentType := app.String(cli.StringOpt{
		Name:   "content-type",
		Value:  "",
		Desc:   "The type of the content the application is able to handle.",
		EnvVar: "CONTENT_TYPE",
	})

	appName := app.String(cli.StringOpt{
		Name:   "appName",
		Value:  "native-ingester",
		Desc:   "The name of the application",
		EnvVar: "APP_NAME",
	})

	app.Action = func() {
		logger.InitDefaultLogger(*appName)

		var collectionsByOriginIds map[string]string
		if err := json.Unmarshal([]byte(*nativeWriterCollectionsByOrigins), &collectionsByOriginIds); err != nil {
			logger.Errorf(nil, err, "Couldn't parse JSON for originId to collection map")
		}

		logger.Infof(nil, "[Startup] Using UUID paths configuration: %# v", *contentUUIDfields)
		bodyParser := native.NewContentBodyParser(*contentUUIDfields)
		writer := native.NewWriter(*nativeWriterAddress, collectionsByOriginIds, *nativeWriterHostHeader, bodyParser)
		logger.Infof(nil, "[Startup] Using native writer configuration: %# v", writer)

		mh := queue.NewMessageHandler(writer, *contentType)

		var messageProducer kafka.Producer
		if *writeQueueAddress != "" {
			messageProducer, err := kafka.NewPerseverantProducer(*writeQueueAddress, *writeQueueTopic, nil, 0, time.Minute)
			if err != nil {
				logger.Errorf(nil, err, "unable to create producer for %v/%v", *writeQueueAddress, *writeQueueTopic)
			}
			logger.Infof(nil, "[Startup] Producer: %# v", messageProducer)
			mh.ForwardTo(messageProducer)
		}

		messageConsumer, err := kafka.NewPerseverantConsumer((*readQueueAddresses)[0], *readQueueGroup, []string{*readQueueTopic}, nil, time.Minute)
		if err != nil {
			logger.Errorf(nil, err, "unable to create message consumer for %v/%v", *readQueueAddresses, *readQueueTopic)
		}

		logger.Infof(nil, "[Startup] Consumer: %# v", messageConsumer)
		logger.Infof(nil, "[Startup] Using native writer configuration: %# v", writer)
		logger.Infof(nil, "[Startup] Using native writer configuration: %# v", *contentUUIDfields)
		if messageProducer != nil {
			logger.Infof(map[string]interface{}{}, "[Startup] Producer: %# v", messageProducer)
		}

		go enableHealthCheck(*port, messageConsumer, messageProducer, writer)
		startMessageConsumption(messageConsumer, mh.HandleMessage)
	}

	err := app.Run(os.Args)
	if err != nil {
		println(err)
	}
}

func enableHealthCheck(port string, consumer kafka.Consumer, producer kafka.Producer, nw native.Writer) {
	hc := resources.NewHealthCheck(consumer, producer, nw)

	r := mux.NewRouter()
	r.HandleFunc("/__health", hc.Handler())
	r.HandleFunc(httphandlers.GTGPath, httphandlers.NewGoodToGoHandler(hc.GTG)).Methods("GET")
	r.HandleFunc(httphandlers.BuildInfoPath, httphandlers.BuildInfoHandler).Methods("GET")
	r.HandleFunc(httphandlers.PingPath, httphandlers.PingHandler).Methods("GET")

	http.Handle("/", r)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		logger.Fatalf(nil, err, "Couldn't set up HTTP listener")
	}
}

func startMessageConsumption(messageConsumer kafka.Consumer, mh func(message kafka.FTMessage) error) {
	var consumerWaitGroup sync.WaitGroup
	consumerWaitGroup.Add(1)

	go func() {
		messageConsumer.StartListening(mh)
		consumerWaitGroup.Done()
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	messageConsumer.Shutdown()
	consumerWaitGroup.Wait()
}
