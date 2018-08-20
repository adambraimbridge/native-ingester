package native

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/native-ingester/config"
	"github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var methodeCollection = []string{"methode"}

const (
	methodeOriginSystemID = "http://cmdb.ft.com/systems/methode-web-pub"

	publishRef   = "tid_test-pub-ref"
	aUUID        = "572d0acc-3f12-4e70-8830-8092c1042a52"
	aTimestamp   = "2017-02-16T12:56:16Z"
	aHash        = "27f79e6d884acdd642d1758c4fd30d43074f8384d552d1ebb1959345"
	aContentType = "application/json; version=1.0"

	withNativeHashHeader    = true
	withoutNativeHashHeader = false
)

var strCollectionsOriginIdsMap = `{
    "config": {
        "http://cmdb.ft.com/systems/methode-web-pub": [
            {
				"content_type": "(application/json).*",
                "collection": "methode"
            }
        ]
    }
}`

var audioStrCollectionsOriginIdsMap = `{
    "config": {
		"http://cmdb.ft.com/systems/next-video-editor": [
            {
                "content_type": "^(application/json).*$",
                "collection": "video"
            },
            {
                "content_type": "^(application/)*(vnd.ft-upp-audio\\+json).*$",
                "collection": "spark"
            }
        ]
    }
}`
var aContentBody = map[string]interface{}{
	"publishReference": publishRef,
	"lastModified":     aTimestamp,
}

func init() {
	logger.InitDefaultLogger("native-ingester")
}

func setupMockNativeWriterService(t *testing.T, status int, hasHash bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if status != 200 {
			w.WriteHeader(status)
		}
		assert.Equal(t, "PUT", req.Method)
		assert.Equal(t, "/"+methodeCollection[0]+"/"+aUUID, req.URL.Path)
		assert.Equal(t, publishRef, req.Header.Get(transactionIDHeader))
		assert.Equal(t, aContentType, req.Header.Get(contentTypeHeader))
		if hasHash {
			assert.Equal(t, aHash, req.Header.Get(nativeHashHeader))
		}
	}))
}

func setupMockNativeWriterGTG(t *testing.T, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if status != 200 {
			w.WriteHeader(status)
		}
		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, httphandlers.GTGPath, req.URL.Path)
	}))
}

func getConfig() (*config.Configuration, error) {
	ior := strings.NewReader(strCollectionsOriginIdsMap)
	return config.ReadConfigFromReader(ior)
}

func getAudioConfig() (*config.Configuration, error) {
	ior := strings.NewReader(audioStrCollectionsOriginIdsMap)
	return config.ReadConfigFromReader(ior)
}

func TestGetCollection(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig()
	assert.NoError(t, err, "It should not return an error")

	w := NewWriter("", *testCollectionsOriginIdsMap, p)

	actualCollection, err := w.GetCollection(methodeOriginSystemID, aContentType)
	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, methodeCollection[0], actualCollection, "It should return the methode collection")

	_, err = w.GetCollection("Origin-Id-that-do-not-exist", aContentType)
	assert.EqualError(t, err, "Origin system not found", "It should return a collection not found error")
	p.AssertExpectations(t)
}

func TestGetVideoCollection(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getAudioConfig()
	assert.NoError(t, err, "It should not return an error")
	w := NewWriter("", *testCollectionsOriginIdsMap, p)
	o := "http://cmdb.ft.com/systems/next-video-editor"

	tests := []struct {
		contentType   string
		expCollection string
		wantErr       bool
	}{
		{
			"application/vnd.ft-upp-audio+json;version=1.0",
			"spark",
			false},
		{
			"application/vnd.ft-upp-audio+json",
			"spark",
			false},
		{
			"application/vnd.ft-upp-audio+json;",
			"spark",
			false},
		{
			"application/vnd.ft-upp-audio+json;version=1.0",
			"spark",
			false},
		{
			"vnd.ft-upp-audio+json",
			"spark",
			false},
		{"application/json",
			"video",
			false},
		{"wrong",
			"",
			true},
	}
	for _, tt := range tests {
		t.Run("Test", func(t *testing.T) {
			actualCollection, err := w.GetCollection(o, tt.contentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestGetVideoCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expCollection != actualCollection {
				t.Errorf("TestGetVideoCollection() = %v, want %v", actualCollection, tt.expCollection)
			}

		})
	}
}

func TestWriteMessageToCollectionWithSuccess(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig()
	assert.NoError(t, err, "It should not return an error")
	p.On("getUUID", aContentBody).Return(aUUID, nil)
	nws := setupMockNativeWriterService(t, 200, withoutNativeHashHeader)
	defer nws.Close()

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef)
	msg.AddContentTypeHeader(aContentType)
	assert.NoError(t, err, "It should not return an error by creating a message")

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p)
	contentUUID, err := w.WriteToCollection(msg, methodeCollection[0])

	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, aUUID, contentUUID)
	p.AssertExpectations(t)
}

func TestWriteMessageWithHashToCollectionWithSuccess(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig()
	assert.NoError(t, err, "It should not return an error")

	p.On("getUUID", aContentBody).Return(aUUID, nil)
	nws := setupMockNativeWriterService(t, 200, withNativeHashHeader)
	defer nws.Close()

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef)
	assert.NoError(t, err, "It should not return an error by creating a message")
	msg.AddHashHeader(aHash)
	msg.AddContentTypeHeader(aContentType)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p)
	contentUUID, err := w.WriteToCollection(msg, methodeCollection[0])

	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, aUUID, contentUUID)
	p.AssertExpectations(t)
}

func TestWriteMessageToCollectionWithContentTypeSuccess(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig()
	assert.NoError(t, err, "It should not return an error")

	p.On("getUUID", aContentBody).Return(aUUID, nil)
	nws := setupMockNativeWriterService(t, 200, withNativeHashHeader)
	defer nws.Close()

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef)
	assert.NoError(t, err, "It should not return an error by creating a message")
	msg.AddHashHeader(aHash)
	msg.AddContentTypeHeader(aContentType)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p)
	contentUUID, err := w.WriteToCollection(msg, methodeCollection[0])

	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, aUUID, contentUUID)
	p.AssertExpectations(t)
}

func TestWriteContentBodyToCollectionFailBecauseOfMissingUUID(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig()
	assert.NoError(t, err, "It should not return an error")

	p.On("getUUID", aContentBody).Return("", errors.New("UUID not found"))
	nws := setupMockNativeWriterService(t, 200, withNativeHashHeader)
	defer nws.Close()

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef)
	assert.NoError(t, err, "It should not return an error by creating a message")
	msg.AddHashHeader(aHash)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p)
	_, err = w.WriteToCollection(msg, methodeCollection[0])

	assert.EqualError(t, err, "UUID not found", "It should return a  UUID not found error")
	p.AssertExpectations(t)
}

func TestWriteContentBodyToCollectionFailBecauseOfNativeRWServiceInternalError(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig()
	assert.NoError(t, err, "It should not return an error")

	p.On("getUUID", aContentBody).Return(aUUID, nil)
	nws := setupMockNativeWriterService(t, 500, withoutNativeHashHeader)
	defer nws.Close()

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef)
	assert.NoError(t, err, "It should not return an error by creating a message")
	msg.AddHashHeader(aHash)
	msg.AddContentTypeHeader(aContentType)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p)
	_, err = w.WriteToCollection(msg, methodeCollection[0])

	assert.EqualError(t, err, "Native writer returned non-200 code", "It should return a non-200 HTTP status error")
	p.AssertExpectations(t)
}

func TestWriteContentBodyToCollectionFailBecauseOfNativeRWServiceNotAvailable(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig()
	assert.NoError(t, err, "It should not return an error")

	p.On("getUUID", aContentBody).Return(aUUID, nil)

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef)
	assert.NoError(t, err, "It should not return an error by creating a message")
	msg.AddHashHeader(aHash)
	msg.AddContentTypeHeader(aContentType)

	w := NewWriter("http://an-address.com", *testCollectionsOriginIdsMap, p)
	_, err = w.WriteToCollection(msg, methodeCollection[0])

	assert.Error(t, err, "It should return an error")
	p.AssertExpectations(t)
}

func TestConnectivityCheckSuccess(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig()
	assert.NoError(t, err, "It should not return an error")

	nws := setupMockNativeWriterGTG(t, 200)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p)
	msg, err := w.ConnectivityCheck()

	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, "Native writer is good to go.", msg, "It should return a positive message")
}

func TestConnectivityCheckSuccessWithoutHostHeader(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig()
	assert.NoError(t, err, "It should not return an error")

	nws := setupMockNativeWriterGTG(t, 200)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p)
	msg, err := w.ConnectivityCheck()

	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, "Native writer is good to go.", msg, "It should return a positive message")
}

func TestConnectivityCheckFailNotGTG(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig()
	assert.NoError(t, err, "It should not return an error")

	nws := setupMockNativeWriterGTG(t, 503)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p)
	msg, err := w.ConnectivityCheck()

	assert.EqualError(t, err, "GTG HTTP status code is 503", "It should return an error")
	assert.Equal(t, "Native writer is not good to go.", msg, "It should return a negative message")
}

func TestConnectivityCheckFailNativeRWServiceNotAvailable(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig()
	assert.NoError(t, err, "It should not return an error")

	w := NewWriter("http://an-address.com", *testCollectionsOriginIdsMap, p)
	msg, err := w.ConnectivityCheck()

	assert.Error(t, err, "It should return an error")
	assert.Equal(t, "Native writer is not good to go.", msg, "It should return a negative message")
}

func TestConnectivityCheckFailToBuildRequest(t *testing.T) {
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig()
	assert.NoError(t, err, "It should not return an error")

	w := NewWriter("http://foo.com  and some spaces", *testCollectionsOriginIdsMap, p)
	msg, err := w.ConnectivityCheck()

	assert.Error(t, err, "It should return an error")
	assert.Equal(t, "Error in building request to check if the native writer is good to go", msg, "It should return a negative message")
}

type ContentBodyParserMock struct {
	mock.Mock
}

func (p *ContentBodyParserMock) getUUID(body map[string]interface{}) (string, error) {
	args := p.Called(body)
	return args.String(0), args.Error(1)
}

func TestBuildNativeMessageSuccess(t *testing.T) {
	msg, err := NewNativeMessage(`{"foo":"bar"}`, aTimestamp, publishRef)
	assert.NoError(t, err, "It should return an error in creating a new message")
	msg.AddHashHeader(aHash)

	assert.Equal(t, msg.body["foo"], "bar", "The body should contain the original attributes")
	assert.Equal(t, msg.body["lastModified"], aTimestamp, "The body should contain the additiona timestamp")
	assert.Equal(t, msg.body["publishReference"], publishRef, "The body should contain the publish reference")
	assert.Equal(t, msg.headers[nativeHashHeader], aHash, "The message should contain the hash")
	assert.Equal(t, msg.TransactionID(), publishRef, "The message should contain the publish reference")
}

func TestBuildNativeMessageFailure(t *testing.T) {
	_, err := NewNativeMessage("__INVALID_BODY__", aTimestamp, publishRef)
	assert.EqualError(t, err, "invalid character '_' looking for beginning of value", "It should return an error in creating a new message")
}
