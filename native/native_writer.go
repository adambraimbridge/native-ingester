package native

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Financial-Times/message-queue-gonsumer/consumer"

	log "github.com/Sirupsen/logrus"
)

// Writer provides the functionalities to write in the native store
type Writer interface {
	Write(msg consumer.Message)
}

type nativeWriter struct {
	address     string
	collections nativeCollections
	hostHeader  string
	httpClient  http.Client
	bodyParser  ContentBodyParser
}

// NewWriter returns a new instance of a native writer
func NewWriter(address string, collectionsOriginIdsMap map[string]string, hostHeader string, parser ContentBodyParser) Writer {
	collections := newNativeCollections(collectionsOriginIdsMap)
	return &nativeWriter{address, collections, hostHeader, http.Client{}, parser}
}

type nativeCollections struct {
	collectionsOriginIdsMap map[string]string
}

func newNativeCollections(collectionsOriginIdsMap map[string]string) nativeCollections {
	return nativeCollections{collectionsOriginIdsMap}
}

func (c nativeCollections) getCollectionByOriginID(originID string) (string, error) {
	collection := c.collectionsOriginIdsMap[originID]
	if collection == "" {
		return "", errors.New("Collection not found")
	}
	return collection, nil
}

func (nw *nativeWriter) Write(msg consumer.Message) {

	pubEvent := publicationEvent{msg}

	cBody, err := pubEvent.contentBody()
	if err != nil {
		log.WithError(err).WithField("transaction_id", pubEvent.transactionID()).Error("Error unmarshalling content body from publication event. Ignoring message.")
		return
	}

	collection, err := nw.collections.getCollectionByOriginID(pubEvent.originSystemID())

	if err != nil {
		log.WithField("transaction_id", pubEvent.transactionID()).WithField("Origin-System-Id", pubEvent.originSystemID()).Info("Skipping content because of not whitelisted Origin-System-Id")
		return
	}

	nw.writeContentBodyToCollection(cBody, collection)

}

func (nw *nativeWriter) writeContentBodyToCollection(cBody contentBody, collection string) {
	contentUUID, err := nw.bodyParser.getUUID(cBody)
	if err != nil {
		log.WithField("transaction_id", cBody.publishReference()).WithError(err).Error("Error extracting uuid. Ignoring message.")
		return
	}
	log.WithField("transaction_id", cBody.publishReference()).WithField("uuid", contentUUID).Info("Start processing native publish event")

	cBodyAsJSON, err := json.Marshal(cBody)

	if err != nil {
		log.WithError(err).WithField("transaction_id", cBody.publishReference()).Error("Error marshalling message")
		return
	}

	requestURL := nw.address + "/" + collection + "/" + contentUUID
	log.WithField("transaction_id", cBody.publishReference()).WithField("requestURL", requestURL).Info("Built request URL for native writer")

	request, err := http.NewRequest("PUT", requestURL, bytes.NewBuffer(cBodyAsJSON))
	if err != nil {
		log.WithError(err).WithField("transaction_id", cBody.publishReference()).WithField("requestURL", requestURL).Error("Error calling native writer. Ignoring message.")
		return
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-Id", cBody.publishReference())

	if len(strings.TrimSpace(nw.hostHeader)) > 0 {
		request.Host = nw.hostHeader
	}

	response, err := nw.httpClient.Do(request)

	if err != nil {
		log.WithError(err).WithField("transaction_id", cBody.publishReference()).WithField("requestURL", requestURL).Error("Error calling native writer. Ignoring message.")
		return
	}
	defer properClose(response)

	if response.StatusCode != http.StatusOK {
		log.WithField("transaction_id", cBody.publishReference()).WithField("responseStatusCode", response.StatusCode).Error("Native writer returned non-200 code")
		return
	}

	log.WithField("transaction_id", cBody.publishReference()).WithField("uuid", contentUUID).Info("Successfully finished processing native publish event")

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
