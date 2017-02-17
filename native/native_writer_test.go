package native

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const methodeOriginSystemID = "http://cmdb.ft.com/systems/methode-web-pub"
const methodeCollection = "methode"
const nativeRWHostHeader = "native-rw"
const publishRef = "tid_test-pub-ref"
const aUUID = "572d0acc-3f12-4e70-8830-8092c1042a52"

var testCollectionsOriginIdsMap = map[string]string{
	methodeOriginSystemID: methodeCollection,
}

var aContentBody = map[string]interface{}{
	"publishReference": publishRef,
}

func setupMockNativeWriterService(t *testing.T, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if status != 200 {
			w.WriteHeader(status)
		}
		assert.Equal(t, "PUT", req.Method)
		assert.Equal(t, "/"+methodeCollection+"/"+aUUID, req.URL.Path)
		assert.Equal(t, nativeRWHostHeader, req.Host)
	}))
}

func setupMockNativeWriterGTG(t *testing.T, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if status != 200 {
			w.WriteHeader(status)
		}
		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, httphandlers.GTGPath, req.URL.Path)
		assert.Equal(t, nativeRWHostHeader, req.Host)
	}))
}

func TestGetCollectionByOriginID(t *testing.T) {
	p := new(ContentBodyParserMock)

	w := NewWriter("", testCollectionsOriginIdsMap, "", p)

	actualCollection, err := w.GetCollectionByOriginID(methodeOriginSystemID)
	assert.Nil(t, err, "It should not return an error")
	assert.Equal(t, methodeCollection, actualCollection, "It should return the methode collection")

	_, err = w.GetCollectionByOriginID("Origin-Id-that-do-not-exist")
	assert.EqualError(t, err, "Collection not found", "It should return a collection not found error")
}

func TestWriteContentBodyToCollectionWithSuccess(t *testing.T) {
	p := new(ContentBodyParserMock)
	p.On("getUUID", aContentBody).Return(aUUID, nil)
	nws := setupMockNativeWriterService(t, 200)
	defer nws.Close()

	w := NewWriter(nws.URL, testCollectionsOriginIdsMap, nativeRWHostHeader, p)
	err := w.WriteContentBodyToCollection(aContentBody, methodeCollection)

	assert.Nil(t, err, "It should not return an error")
	p.AssertExpectations(t)
}

func TestWriteContentBodyToCollectionFailBecauseOfMissingUUID(t *testing.T) {
	p := new(ContentBodyParserMock)
	p.On("getUUID", aContentBody).Return("", errors.New("UUID not found"))
	nws := setupMockNativeWriterService(t, 200)
	defer nws.Close()

	w := NewWriter(nws.URL, testCollectionsOriginIdsMap, nativeRWHostHeader, p)
	err := w.WriteContentBodyToCollection(aContentBody, methodeCollection)

	assert.EqualError(t, err, "UUID not found", "It should return a  UUID not found error")
	p.AssertExpectations(t)
}

func TestWriteContentBodyToCollectionFailBecauseOfNativeRWServiceInternalError(t *testing.T) {
	p := new(ContentBodyParserMock)
	p.On("getUUID", aContentBody).Return(aUUID, nil)
	nws := setupMockNativeWriterService(t, 500)
	defer nws.Close()

	w := NewWriter(nws.URL, testCollectionsOriginIdsMap, nativeRWHostHeader, p)
	err := w.WriteContentBodyToCollection(aContentBody, methodeCollection)

	assert.EqualError(t, err, "Native writer returned non-200 code", "It should return a non-200 HTTP status error")
	p.AssertExpectations(t)
}

func TestWriteContentBodyToCollectionFailBecauseOfNativeRWServiceNotAvailable(t *testing.T) {
	p := new(ContentBodyParserMock)
	p.On("getUUID", aContentBody).Return(aUUID, nil)

	w := NewWriter("http://an-address.com", testCollectionsOriginIdsMap, nativeRWHostHeader, p)
	err := w.WriteContentBodyToCollection(aContentBody, methodeCollection)

	assert.Error(t, err, "It should return an error")
	p.AssertExpectations(t)
}

func TestConnectivityCheckSuccess(t *testing.T) {
	p := new(ContentBodyParserMock)

	nws := setupMockNativeWriterGTG(t, 200)

	w := NewWriter(nws.URL, testCollectionsOriginIdsMap, nativeRWHostHeader, p)
	msg, err := w.ConnectivityCheck()

	assert.Nil(t, err, "It should not return an error")
	assert.Equal(t, "Native writer is good to go.", msg, "It should return a positive message")
}

func TestConnectivityCheckFailNotGTG(t *testing.T) {
	p := new(ContentBodyParserMock)

	nws := setupMockNativeWriterGTG(t, 503)

	w := NewWriter(nws.URL, testCollectionsOriginIdsMap, nativeRWHostHeader, p)
	msg, err := w.ConnectivityCheck()

	assert.EqualError(t, err, "GTG HTTP status code is 503", "It should return an error")
	assert.Equal(t, "Native writer is not good to go.", msg, "It should return a negative message")
}

func TestConnectivityCheckFailNativeRWServiceNotAvailable(t *testing.T) {
	p := new(ContentBodyParserMock)

	w := NewWriter("http://an-address.com", testCollectionsOriginIdsMap, nativeRWHostHeader, p)
	msg, err := w.ConnectivityCheck()

	assert.Error(t, err, "It should return an error")
	assert.Equal(t, "Native writer is not good to go.", msg, "It should return a negative message")
}

func TestConnectivityCheckFailToBuildRequest(t *testing.T) {
	p := new(ContentBodyParserMock)

	w := NewWriter("http://foo.com  and some spaces", testCollectionsOriginIdsMap, nativeRWHostHeader, p)
	msg, err := w.ConnectivityCheck()

	assert.Error(t, err, "It should return an error")
	assert.Equal(t, "Error in building request to check if the native writer is good to go", msg, "It should return a negative message")
}

type ContentBodyParserMock struct {
	mock.Mock
}

func (p ContentBodyParserMock) getUUID(body map[string]interface{}) (string, error) {
	args := p.Called(body)
	return args.String(0), args.Error(1)
}
