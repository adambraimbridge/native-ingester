package queue

import (
	"encoding/json"
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

func (pe *publicationEvent) contentBody() (native.ContentBody, error) {
	body := make(native.ContentBody)
	if err := json.Unmarshal([]byte(pe.Body), &body); err != nil {
		return native.ContentBody{}, err
	}

	timestamp, found := pe.Headers["Message-Timestamp"]
	if !found {
		return native.ContentBody{}, errors.New("Publish event does not contain timestamp")
	}
	body["lastModified"] = timestamp
	body["publishReference"] = pe.transactionID()

	return body, nil
}

func (pe *publicationEvent) producerMsg() producer.Message {
	return producer.Message{
		Headers: pe.Headers,
		Body:    pe.Body,
	}
}
