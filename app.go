package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
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

	"context"

	"github.com/golang/go/src/pkg/fmt"
	"pack.ag/amqp"
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
		Desc:   "Zookeeper addresses (host:port) to connect to the consumer queue.",
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
		Desc:   "Address (URL) of service that writes persistently the native content",
		EnvVar: "NATIVE_RW_ADDRESS",
	})
	nativeWriterCollectionsByOrigins := app.String(cli.StringOpt{
		Name:   "native-writer-collections-by-origins",
		Value:  "{}",
		Desc:   "Mapping from originId (URI) to native collection name, in JSON format. e.g. {\"http://cmdb.ft.com/systems/methode-web-pub\":\"methode\"}",
		EnvVar: "NATIVE_RW_COLLECTIONS_BY_ORIGINS",
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
		Desc:   "Kafka address (host:port) to connect to the producer queue.",
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
		Desc:   "The type of the content (for logging purposes, e.g. \"Content\" or \"Annotations\") the application is able to handle.",
		EnvVar: "CONTENT_TYPE",
	})

	appName := app.String(cli.StringOpt{
		Name:   "appName",
		Value:  "native-ingester",
		Desc:   "The name of the application",
		EnvVar: "APP_NAME",
	})

	mqUser := app.String(cli.StringOpt{
		Name:   "mqUser",
		Value:  "",
		Desc:   "AmazonMQ broker username",
		EnvVar: "MQ_USER",
	})

	mqPassword := app.String(cli.StringOpt{
		Name:   "mqPassword",
		Value:  "",
		Desc:   "AmazonMQ broker password",
		EnvVar: "MQ_PASSWORD",
	})
	mqEndpoint := app.String(cli.StringOpt{
		Name:   "mqEndpoint",
		Value:  "",
		Desc:   "AmazonMQ broker STOMP endpoint",
		EnvVar: "MQ_ENDPOINT",
	})
	app.Action = func() {
		logger.InitDefaultLogger(*appName)

		var collectionsByOriginIds map[string]string
		if err := json.Unmarshal([]byte(*nativeWriterCollectionsByOrigins), &collectionsByOriginIds); err != nil {
			logger.Errorf(nil, err, "Couldn't parse JSON for originId to collection map")
		}

		logger.Infof(nil, "[Startup] Using UUID paths configuration: %# v", *contentUUIDfields)
		bodyParser := native.NewContentBodyParser(*contentUUIDfields)
		writer := native.NewWriter(*nativeWriterAddress, collectionsByOriginIds, bodyParser)
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
		activeMQStop := make(chan bool)
		go startActiveMQMessageConsumption(*mqUser, *mqPassword, *mqEndpoint, *readQueueTopic, activeMQStop)
		startMessageConsumption(messageConsumer, mh.HandleMessage)
		activeMQStop <- true
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
	messageConsumer.StartListening(mh)

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	messageConsumer.Shutdown()
}

func startActiveMQMessageConsumption(user, password, endpoint, topic string, stop chan bool) {
	if len(endpoint) == 0 {
		logger.Infof(nil, "[mq] ActiveMQ Consumer not configured")
		return
	}

	// Create client
	client, err := amqp.Dial(endpoint, amqp.ConnSASLPlain(user, password))
	if err != nil {
		logger.Errorf(nil, err, "Error dialing AMQP server.")
		return
	}
	defer client.Close()

	// Open a session
	session, err := client.NewSession()
	if err != nil {
		logger.Errorf(nil, err, "Error creating AMQP session.")
		return
	}

	ctx := context.Background()

	logger.Infof(nil, "[mq] Waiting for message from ActiveMQ topic [%s]", topic)

	// Create a receiver
	receiver, err := session.NewReceiver(
		amqp.LinkSourceAddress(topic),
		amqp.LinkCredit(10),
	)
	if err != nil {
		logger.Errorf(nil, err, "[mq] Error creating receiver link.")
		return
	}
	defer func() {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		receiver.Close(ctx)
		cancel()
	}()

	for {
		// Receive next message
		msg, err := receiver.Receive(ctx)
		if err != nil {
			logger.Errorf(nil, err, "[mq] Reading message from AMQP:", err)
			return
		}

		// Accept message
		msg.Accept()
		fmt.Printf("[mq] Message received: %v, %v \n", len(msg.Data))

	}

	logger.Infof(nil, "[mq] Finished reading messages")

}
