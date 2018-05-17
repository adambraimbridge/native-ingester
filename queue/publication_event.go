package queue

import (
	"errors"
	"strings"

	"github.com/Financial-Times/kafka-client-go/kafka"
	"github.com/Financial-Times/native-ingester/native"
)

type publicationEvent struct {
	kafka.FTMessage
}

func (pe *publicationEvent) transactionID() string {
	return pe.Headers["X-Request-Id"]
}

func (pe *publicationEvent) originSystemID() string {
	return strings.TrimSpace(pe.Headers["Origin-System-Id"])
}

func (pe *publicationEvent) nativeMessage() (native.NativeMessage, error) {

	timestamp, found := pe.Headers["Message-Timestamp"]
	if !found {
		return native.NativeMessage{}, errors.New("Publish event does not contain timestamp")
	}

	msg, err := native.NewNativeMessage(pe.Body, timestamp, pe.transactionID())

	if err != nil {
		return native.NativeMessage{}, err
	}

	nativeHash, found := pe.Headers["Native-Hash"]
	if found {
		msg.AddHashHeader(nativeHash)
	}

	contentType, found := pe.Headers["Content-Type"]
	if found {
		msg.AddContentTypeHeader(contentType)
	}

	return msg, nil
}

func (pe *publicationEvent) producerMsg() kafka.FTMessage {
	return kafka.FTMessage{
		Headers: pe.Headers,
		Body:    pe.Body,
	}
}
