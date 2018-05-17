package native

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/Financial-Times/go-logger"

	"github.com/Financial-Times/service-status-go/httphandlers"
)

const nativeHashHeader = "X-Native-Hash"
const contentTypeHeader = "Content-Type"
const transactionIDHeader = "X-Request-Id"

// Writer provides the functionalities to write in the native store
type Writer interface {
	GetCollectionByOriginID(originID string) (string, error)
	WriteToCollection(msg NativeMessage, collection string) (string, error)
	ConnectivityCheck() (string, error)
}

type nativeWriter struct {
	address     string
	collections nativeCollections
	httpClient  http.Client
	bodyParser  ContentBodyParser
}

// NewWriter returns a new instance of a native writer
func NewWriter(address string, collectionsOriginIdsMap map[string]string, parser ContentBodyParser) Writer {
	collections := newNativeCollections(collectionsOriginIdsMap)
	return &nativeWriter{address, collections, http.Client{}, parser}
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

func (nw *nativeWriter) WriteToCollection(msg NativeMessage, collection string) (string, error) {
	contentUUID, err := nw.bodyParser.getUUID(msg.body)
	if err != nil {
		logger.NewEntry(msg.transactionID()).WithError(err).Error("Error extracting uuid. Ignoring message.")
		return contentUUID, err
	}
	logger.NewEntry(msg.transactionID()).WithUUID(contentUUID).Info("Start processing native publish event")
	cBodyAsJSON, err := json.Marshal(msg.body)

	if err != nil {
		logger.NewEntry(msg.transactionID()).WithUUID(contentUUID).WithError(err).Error("Error marshalling message")
		return contentUUID, err
	}

	requestURL := nw.address + "/" + collection + "/" + contentUUID
	request, err := http.NewRequest("PUT", requestURL, bytes.NewBuffer(cBodyAsJSON))
	if err != nil {
		logger.NewEntry(msg.transactionID()).WithUUID(contentUUID).WithError(err).Error("Error calling native writer. Ignoring message.")
		return contentUUID, err
	}

	for header, value := range msg.headers {
		request.Header.Set(header, value)
	}

	if request.Header.Get(contentTypeHeader) == "" {
		logger.NewEntry(msg.transactionID()).
			WithUUID(contentUUID).
			Warn("Native-save request does not have content-type header, defaulting to application/json.")

		request.Header.Set("Content-Type", "application/json")

	}

	response, err := nw.httpClient.Do(request)

	if err != nil {
		logger.NewEntry(msg.transactionID()).WithUUID(contentUUID).WithError(err).Error("Error calling native writer. Ignoring message.")
		return contentUUID, err
	}
	defer properClose(msg.transactionID(), response)

	if isNot2XXStatusCode(response.StatusCode) {
		errMsg := "Native writer returned non-200 code"
		err := errors.New(errMsg)
		logger.NewEntry(msg.transactionID()).WithUUID(contentUUID).WithError(err).Error(errMsg)
		return contentUUID, err
	}

	logger.NewEntry(msg.transactionID()).WithUUID(contentUUID).Info("Successfully finished processing native publish event")
	return contentUUID, nil
}

func properClose(tid string, resp *http.Response) {
	_, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		logger.NewEntry(tid).WithError(err).Warn("Couldn't read response body")
	}
	err = resp.Body.Close()
	if err != nil {
		logger.NewEntry(tid).WithError(err).Warn("Couldn't close response body")
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
	resp, err := nw.httpClient.Do(req)
	if err != nil {
		return "Native writer is not good to go.", err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return "Native writer is not good to go.", fmt.Errorf("GTG HTTP status code is %v", resp.StatusCode)
	}
	return "Native writer is good to go.", nil
}

// NativeMessage is the message accepted by the native writer
type NativeMessage struct {
	body    map[string]interface{}
	headers map[string]string
}

// NewNativeMessage returns a new instance of a NativeMessage
func NewNativeMessage(contentBody string, timestamp string, transactionID string) (NativeMessage, error) {
	body := make(map[string]interface{})
	if err := json.Unmarshal([]byte(contentBody), &body); err != nil {
		return NativeMessage{}, err
	}

	body["lastModified"] = timestamp
	body["publishReference"] = transactionID

	msg := NativeMessage{body, make(map[string]string)}
	msg.headers[transactionIDHeader] = transactionID

	return msg, nil
}

//AddHashHeader adds the hash of the native content as a header
func (msg *NativeMessage) AddHashHeader(hash string) {
	msg.headers[nativeHashHeader] = hash
}

//AddContentTypeHeader adds the content type of the native content as a header
func (msg *NativeMessage) AddContentTypeHeader(hash string) {
	msg.headers[contentTypeHeader] = hash
}

func (msg *NativeMessage) transactionID() string {
	return msg.headers[transactionIDHeader]
}
