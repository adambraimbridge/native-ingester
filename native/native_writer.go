package native

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Financial-Times/service-status-go/httphandlers"
	log "github.com/Sirupsen/logrus"
)

// Writer provides the functionalities to write in the native store
type Writer interface {
	GetCollectionByOriginID(originID string) (string, error)
	WriteContentBodyToCollection(cBody ContentBody, collection string) error
	ConnectivityCheck() (string, error)
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

func (nw *nativeWriter) GetCollectionByOriginID(originID string) (string, error) {
	return nw.collections.getCollectionByOriginID(originID)
}

func (nw *nativeWriter) WriteContentBodyToCollection(cBody ContentBody, collection string) error {
	contentUUID, err := nw.bodyParser.getUUID(cBody)
	if err != nil {
		log.WithField("transaction_id", cBody.publishReference()).WithError(err).Error("Error extracting uuid. Ignoring message.")
		return err
	}
	log.WithField("transaction_id", cBody.publishReference()).WithField("uuid", contentUUID).Info("Start processing native publish event")

	cBodyAsJSON, err := json.Marshal(cBody)

	if err != nil {
		log.WithError(err).WithField("transaction_id", cBody.publishReference()).Error("Error marshalling message")
		return err
	}

	requestURL := nw.address + "/" + collection + "/" + contentUUID
	log.WithField("transaction_id", cBody.publishReference()).WithField("requestURL", requestURL).Info("Built request URL for native writer")

	request, err := http.NewRequest("PUT", requestURL, bytes.NewBuffer(cBodyAsJSON))
	if err != nil {
		log.WithError(err).WithField("transaction_id", cBody.publishReference()).WithField("requestURL", requestURL).Error("Error calling native writer. Ignoring message.")
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-Id", cBody.publishReference())

	if len(strings.TrimSpace(nw.hostHeader)) > 0 {
		request.Host = nw.hostHeader
	}

	response, err := nw.httpClient.Do(request)

	if err != nil {
		log.WithError(err).WithField("transaction_id", cBody.publishReference()).WithField("requestURL", requestURL).Error("Error calling native writer. Ignoring message.")
		return err
	}
	defer properClose(response)

	if isNot2XXStatusCode(response.StatusCode) {
		log.WithField("transaction_id", cBody.publishReference()).WithField("responseStatusCode", response.StatusCode).Error("Native writer returned non-200 code")
		return errors.New("Native writer returned non-200 code")
	}

	log.WithField("transaction_id", cBody.publishReference()).WithField("uuid", contentUUID).Info("Successfully finished processing native publish event")
	return nil
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

func isNot2XXStatusCode(statusCode int) bool {
	return statusCode < 200 || statusCode >= 300
}

func (nw nativeWriter) ConnectivityCheck() (string, error) {
	req, err := http.NewRequest("GET", nw.address+httphandlers.GTGPath, nil)
	if err != nil {
		return "Error in building request to check if the native writer is good to go", err
	}
	req.Host = nw.hostHeader

	resp, err := nw.httpClient.Do(req)
	if err != nil {
		return "Native writer is not good to go.", err
	}
	if resp.StatusCode != http.StatusOK {
		return "Native writer is not good to go.", fmt.Errorf("GTG HTTP status code is %v", resp.StatusCode)
	}
	return "Native writer is good to go.", nil
}
