package queue

import (
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/native-ingester/native"

	log "github.com/Sirupsen/logrus"
	"context"
	"github.com/opentracing/opentracing-go"
)

// MessageHandler handles messages consumed from a queue
type MessageHandler struct {
	ctx context.Context
	writer   native.Writer
	producer producer.MessageProducer
	forwards bool
}

// NewMessageHandler returns a new instance of MessageHandler
func NewMessageHandler(w native.Writer) *MessageHandler {
	return &MessageHandler{ ctx: context.Background(), writer: w}
}

// HandleMessage implements the strategy for handling message from a queue
func (mh *MessageHandler) HandleMessage(msg consumer.Message) {

	defer mh.ctx.Done()

	span, ctx := opentracing.StartSpanFromContext(mh.ctx, "handle-message")
	defer span.Finish()

	pubEvent := publicationEvent{msg}

	cBody, err := pubEvent.contentBody()
	if err != nil {
		log.WithError(err).WithField("transaction_id", pubEvent.transactionID()).Error("Error unmarshalling content body from publication event. Ignoring message.")
		return
	}

	collection, err := mh.writer.GetCollectionByOriginID(pubEvent.originSystemID())
	if err != nil {
		log.WithField("transaction_id", pubEvent.transactionID()).WithField("Origin-System-Id", pubEvent.originSystemID()).Info("Skipping content because of not whitelisted Origin-System-Id")
		return
	}

	writerErr := mh.writer.WriteContentBodyToCollection(ctx, cBody, collection)
	if writerErr != nil {
		log.WithError(writerErr).WithField("transaction_id", pubEvent.transactionID()).Error("Failed to write native content")
		return
	}

	if mh.forwards {

		mh.forward(ctx, &pubEvent)
	}
}

func (mh *MessageHandler) forward(ctx context.Context, pe *publicationEvent){
	if span := opentracing.SpanFromContext(ctx); span != nil {
		sp := opentracing.StartSpan(
			"write-pub-event",
			opentracing.ChildOf(span.Context()))
		defer sp.Finish()
	}
	log.WithField("transaction_id", pe.transactionID()).Info("Forwarding consumed message to different queue...")
	forwardErr := mh.producer.SendMessage("", pe.producerMsg())
	if forwardErr != nil {
		log.WithError(forwardErr).WithField("transaction_id", pe.transactionID()).Error("Failed to forward consumed message to a different queue")
		return
	}
	log.WithField("transaction_id", pe.transactionID()).Info("Consumed message successfully forwarded")
}

// ForwardTo sets up the message producer to forward messages after writing in the native store
func (mh *MessageHandler) ForwardTo(p producer.MessageProducer) {
	mh.producer = p
	mh.forwards = true
}
