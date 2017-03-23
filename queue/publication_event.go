package queue

import (
	"errors"
	"strings"

	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/native-ingester/native"
)

type publicationEvent struct {
	consumer.Message
}

func (pe *publicationEvent) transactionID() string {
	return pe.Headers["X-Request-Id"]
}

func (pe *publicationEvent) originSystemID() string {
	return strings.TrimSpace(pe.Headers["Origin-System-Id"])
}

func (pe *publicationEvent) nativeWriterMessage() (native.WriterMessage, error) {

	timestamp, found := pe.Headers["Message-Timestamp"]
	if !found {
		return native.WriterMessage{}, errors.New("Publish event does not contain timestamp")
	}

	nativeHash := pe.Headers["Native-Hash"]

	msg, err := native.NewWriterMessage(pe.Body, timestamp, nativeHash, pe.transactionID())

	if err != nil {
		return native.WriterMessage{}, err
	}

	return msg, nil
}

func (pe *publicationEvent) producerMsg() producer.Message {
	return producer.Message{
		Headers: pe.Headers,
		Body:    pe.Body,
	}
}
