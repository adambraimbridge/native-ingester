package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/jmoiron/jsonq"

	log "github.com/Sirupsen/logrus"
)

type messageHandler struct {
	uuidJSONPaths []string
	writerConfig  NativeWriterConfig
	httpClient    http.Client
}

func newMessageHandler(uuidPaths []string, config NativeWriterConfig) *messageHandler {
	return &messageHandler{
		uuidJSONPaths: uuidPaths,
		writerConfig:  config,
		httpClient:    http.Client{},
	}
}

func (mh *messageHandler) handleMessage(msg consumer.Message) {
	tid := msg.Headers["X-Request-Id"]
	coll := mh.writerConfig.CollectionsByOriginIds[strings.TrimSpace(msg.Headers["Origin-System-Id"])]
	if coll == "" {
		log.WithField("transaction_id", tid).WithField("Origin-System-Id", msg.Headers["Origin-System-Id"]).Info("Skipping content because of not whitelisted Origin-System-Id")
		return
	}
	contents := make(map[string]interface{})
	if err := json.Unmarshal([]byte(msg.Body), &contents); err != nil {
		log.WithError(err).WithField("transaction_id", tid).Error("Error unmarshalling message. Ignoring message.")
		return
	}

	uuid := extractUUID(contents, mh.uuidJSONPaths)
	if uuid == "" {
		log.WithField("transaction_id", tid).Error("Error extracting uuid. Ignoring message.")
		return
	}
	log.WithField("transaction_id", tid).WithField("uuid", uuid).Info("Start processing native publish event")

	requestURL := mh.writerConfig.Address + "/" + coll + "/" + uuid
	log.WithField("transaction_id", tid).WithField("requestURL", requestURL).Info("Built request URL for native writer")

	contents["lastModified"] = time.Now().Round(time.Millisecond)

	bodyWithTimestamp, err := json.Marshal(contents)

	if err != nil {
		log.WithError(err).WithField("transaction_id", tid).Error("Error marshalling message")
		return
	}

	request, err := http.NewRequest("PUT", requestURL, bytes.NewBuffer(bodyWithTimestamp))
	if err != nil {
		log.WithError(err).WithField("transaction_id", tid).WithField("requestURL", requestURL).Error("Error calling native writer. Ignoring message.")
		return
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-Id", tid)
	if len(strings.TrimSpace(mh.writerConfig.Header)) > 0 {
		request.Host = mh.writerConfig.Header
	}

	response, err := mh.httpClient.Do(request)

	if err != nil {
		log.WithError(err).WithField("transaction_id", tid).WithField("requestURL", requestURL).Error("Error calling native writer. Ignoring message.")
		return
	}
	defer properClose(response)

	if response.StatusCode != http.StatusOK {
		log.WithField("transaction_id", tid).WithField("responseStatusCode", response.StatusCode).Error("Native writer returned non-200 code")
		return
	}

	log.WithField("transaction_id", tid).WithField("uuid", uuid).Info("Successfully finished processing native publish event")
}

func extractUUID(contents map[string]interface{}, uuidFields []string) string {
	jq := jsonq.NewQuery(contents)
	for _, uuidField := range uuidFields {
		uuid, err := jq.String(strings.Split(uuidField, ".")...)
		if err == nil && uuid != "" {
			return uuid
		}
	}
	return ""
}

func properClose(resp *http.Response) {
	_, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		log.WithError(err).Warn("Couldn't read response body")
	}
	err = resp.Body.Close()
	if err != nil {
		log.WithError(err).Warn("Couldn't close response body")
	}
}
