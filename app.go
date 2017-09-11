package main

import (
	"encoding/json"
	"github.com/Financial-Times/go-logger"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
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
	readQueueHostHeader := app.String(cli.StringOpt{
		Name:   "read-queue-host-header",
		Value:  "kafka",
		Desc:   "The host header for the queue to read the messages from.",
		EnvVar: "Q_READ_HOST_HEADER",
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
		Desc:   "The topic to write the messages to.",
		EnvVar: "Q_WRITE_TOPIC",
	})
	writeQueueHostHeader := app.String(cli.StringOpt{
		Name:   "write-queue-host-header",
		Value:  "kafka",
		Desc:   "The host header for the queue to write the messages to.",
		EnvVar: "Q_WRITE_HOST_HEADER",
	})

	appName := app.String(cli.StringOpt{
		Name:   "appName",
		Value:  "native-ingester",
		Desc:   "The name of the application",
		EnvVar: "APP_NAME",
	})

	app.Action = func() {
		httpClient := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				MaxIdleConnsPerHost:   20,
				TLSHandshakeTimeout:   3 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		}

		srcConf := consumer.QueueConfig{
			Addrs:                *readQueueAddresses,
			Group:                *readQueueGroup,
			Topic:                *readQueueTopic,
			Queue:                *readQueueHostHeader,
			ConcurrentProcessing: false,
		}
		logger.InitDefaultLogger(*appName)
		var collectionsByOriginIds map[string]string
		if err := json.Unmarshal([]byte(*nativeWriterCollectionsByOrigins), &collectionsByOriginIds); err != nil {
			logger.Errorf(nil, err, "Couldn't parse JSON for originId to collection map")
		}

		logger.Infof(nil, "[Startup] Using UUID paths configuration: %# v", *contentUUIDfields)
		bodyParser := native.NewContentBodyParser(*contentUUIDfields)
		writer := native.NewWriter(*nativeWriterAddress, collectionsByOriginIds, *nativeWriterHostHeader, bodyParser)
		logger.Infof(nil, "[Startup] Using native writer configuration: %# v", writer)

		mh := queue.NewMessageHandler(writer)

		var messageProducer producer.MessageProducer
		var producerConfig *producer.MessageProducerConfig
		if *writeQueueAddress != "" {
			producerConfig = &producer.MessageProducerConfig{
				Addr:  *writeQueueAddress,
				Topic: *writeQueueTopic,
				Queue: *writeQueueHostHeader,
			}
			messageProducer = producer.NewMessageProducerWithHTTPClient(*producerConfig, httpClient)
			logger.Infof(nil, "[Startup] Producer: %# v", messageProducer)
			mh.ForwardTo(messageProducer)
		}

		messageConsumer := consumer.NewConsumer(srcConf, mh.HandleMessage, httpClient)
		logger.Infof(nil, "[Startup] Consumer: %# v", messageConsumer)
		logger.Infof(nil, "[Startup] Using source configuration: %# v", srcConf)
		logger.Infof(nil, "[Startup] Using native writer configuration: %# v", writer)
		logger.Infof(nil, "[Startup] Using native writer configuration: %# v", *contentUUIDfields)
		if messageProducer != nil {
			logger.Infof(map[string]interface{}{}, "[Startup] Producer: %# v", messageProducer)
		}

		go enableHealthCheck(*port, messageConsumer, messageProducer, writer)
		startMessageConsumption(messageConsumer)
	}

	err := app.Run(os.Args)
	if err != nil {
		println(err)
	}
}

func enableHealthCheck(port string, consumer consumer.MessageConsumer, producer producer.MessageProducer, nw native.Writer) {
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
