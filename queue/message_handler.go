package queue

import (
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/native-ingester/native"

	log "github.com/Sirupsen/logrus"
)

// MessageHandler handles messages consumed from a queue
type MessageHandler struct {
	writer   native.Writer
	producer producer.MessageProducer
	forwards bool
}

// NewMessageHandler returns a new instance of MessageHandler
func NewMessageHandler(w native.Writer) *MessageHandler {
	return &MessageHandler{writer: w}
}

// HandleMessage implements the strategy for handling message from a queue
func (mh *MessageHandler) HandleMessage(msg consumer.Message) {
	pubEvent := publicationEvent{msg}

	writerMsg, err := pubEvent.nativeMessage()
	if err != nil {
		log.WithError(err).WithField("transaction_id", pubEvent.transactionID()).Error("Error unmarshalling content body from publication event. Ignoring message.")
		return
	}

	collection, err := mh.writer.GetCollectionByOriginID(pubEvent.originSystemID())
	if err != nil {
		log.WithField("transaction_id", pubEvent.transactionID()).WithField("Origin-System-Id", pubEvent.originSystemID()).Info("Skipping content because of not whitelisted Origin-System-Id")
		return
	}

	writerErr := mh.writer.WriteToCollection(writerMsg, collection)
	if writerErr != nil {
		log.WithError(writerErr).WithField("transaction_id", pubEvent.transactionID()).Error("Failed to write native content")
		return
	}

	if mh.forwards {
		log.WithField("transaction_id", pubEvent.transactionID()).Info("Forwarding consumed message to different queue...")
		forwardErr := mh.producer.SendMessage("", pubEvent.producerMsg())
		if forwardErr != nil {
			log.WithError(forwardErr).WithField("transaction_id", pubEvent.transactionID()).Error("Failed to forward consumed message to a different queue")
			return
		}
		log.WithField("transaction_id", pubEvent.transactionID()).Info("Consumed message successfully forwarded")
	}
}

// ForwardTo sets up the message producer to forward messages after writing in the native store
func (mh *MessageHandler) ForwardTo(p producer.MessageProducer) {
	mh.producer = p
	mh.forwards = true
}
