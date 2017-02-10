package native

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
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

func (pe *publicationEvent) contentBody() (contentBody, error) {
	body := make(contentBody)
	if err := json.Unmarshal([]byte(pe.Body), &body); err != nil {
		return contentBody{}, err
	}

	timestamp, found := pe.Headers["Message-Timestamp"]
	if !found {
		return contentBody{}, errors.New("Publish event does not contain timestamp")
	}
	body["lastModified"] = timestamp
	body["publishReference"] = pe.transactionID()

	return body, nil
}

type contentBody map[string]interface{}

func (body contentBody) publishReference() string {
	return body["lastModified"].(string)
}
