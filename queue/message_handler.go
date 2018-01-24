package queue

import (
	"fmt"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/native-ingester/native"
)

// MessageHandler handles messages consumed from a queue
type MessageHandler struct {
	writer      native.Writer
	producer    producer.MessageProducer
	forwards    bool
	contentType string
}

// NewMessageHandler returns a new instance of MessageHandler
func NewMessageHandler(w native.Writer, contentType string) *MessageHandler {
	return &MessageHandler{writer: w, contentType: contentType}
}

// HandleMessage implements the strategy for handling message from a queue
func (mh *MessageHandler) HandleMessage(msg consumer.Message) {
	pubEvent := publicationEvent{msg}

	writerMsg, err := pubEvent.nativeMessage()
	if err != nil {
		logger.NewMonitoringEntry("Ingest", pubEvent.transactionID(), mh.contentType).
			WithError(err).
			Error("Error unmarshalling content body from publication event. Ignoring message.")
		return
	}

	collection, err := mh.writer.GetCollectionByOriginID(pubEvent.originSystemID())
	if err != nil {
		logger.NewMonitoringEntry("Ingest", pubEvent.transactionID(), mh.contentType).
			WithValidFlag(false).
			Warn(fmt.Sprintf("Skipping content because of not whitelisted Origin-System-Id: %s", pubEvent.originSystemID()))
		return
	}

	contentUUID, writerErr := mh.writer.WriteToCollection(writerMsg, collection)
	if writerErr != nil {
		logger.NewMonitoringEntry("Ingest", pubEvent.transactionID(), mh.contentType).
			WithError(writerErr).
			Error("Failed to write native content")
		return
	}

	if mh.forwards {
		logger.NewEntry(pubEvent.transactionID()).Info("Forwarding consumed message to different queue")
		forwardErr := mh.producer.SendMessage("", pubEvent.producerMsg())
		if forwardErr != nil {
			logger.NewMonitoringEntry("Ingest", pubEvent.transactionID(), mh.contentType).
				WithUUID(contentUUID).
				WithError(forwardErr).
				Error("Failed to forward consumed message to a different queue")
			return
		}
		logger.NewMonitoringEntry("Ingest", pubEvent.transactionID(), mh.contentType).
			WithUUID(contentUUID).
			Info("Successfully ingested")
	}
}

// ForwardTo sets up the message producer to forward messages after writing in the native store
func (mh *MessageHandler) ForwardTo(p producer.MessageProducer) {
	mh.producer = p
	mh.forwards = true
}
