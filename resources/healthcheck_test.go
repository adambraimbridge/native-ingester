package resources

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/native-ingester/native"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"context"
)

func TestHappyHealthCheckWithoutProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	hc := NewHealthCheck(c, nw, nil)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueProxyReachable","ok":true`, "Consumer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":true`, "Native writer healthcheck should be happy")
	assert.NotContains(t, w.Body.String(), `"name":"ProducerQueueProxyReachable","ok":`, "Producer healthcheck should not appear")
}

func TestHappyHealthCheckWithProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("I'm a happy producer", nil)
	hc := NewHealthCheck(c, nw, p)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueProxyReachable","ok":true`, "Consumer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":true`, "Native writer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"ProducerQueueProxyReachable","ok":true`, "Producer healthcheck should be happy")
}

func TestUnhappyConsumerHealthCheckWithoutProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm an unhappy consumer", errors.New("Screw you guys I'm going home!"))
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	hc := NewHealthCheck(c, nw, nil)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueProxyReachable","ok":false`, "Consumer healthcheck should be unhappy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":true`, "Native writer healthcheck should be happy")
	assert.NotContains(t, w.Body.String(), `"name":"ProducerQueueProxyReachable","ok":`, "Producer healthcheck should not appear")
}

func TestUnhappyConsumerHealthCheckWithProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm an unhappy consumer", errors.New("Screw you guys I'm going home!"))
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("I'm a happy producer", nil)
	hc := NewHealthCheck(c, nw, p)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueProxyReachable","ok":false`, "Consumer healthcheck should be unhappy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":true`, "Native writer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"ProducerQueueProxyReachable","ok":true`, "Producer healthcheck should be happy")
}

func TestUnhappyNativeWriterHealthCheckWithoutProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an unhappy writer", errors.New("Oh, my God, they killed Kenny!"))
	hc := NewHealthCheck(c, nw, nil)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueProxyReachable","ok":true`, "Consumer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":false`, "Native writer healthcheck should be unhappy")
	assert.NotContains(t, w.Body.String(), `"name":"ProducerQueueProxyReachable","ok":`, "Producer healthcheck should not appear")
}

func TestUnhappyNativeWriterHealthCheckWithProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an unhappy writer", errors.New("Oh, my God, they killed Kenny!"))
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("I'm a happy producer", nil)
	hc := NewHealthCheck(c, nw, p)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueProxyReachable","ok":true`, "Consumer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":false`, "Native writer healthcheck should be unhappy")
	assert.Contains(t, w.Body.String(), `"name":"ProducerQueueProxyReachable","ok":true`, "Producer healthcheck should be happy")
}

func TestUnhappyProducerHealthCheck(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an happy writer", nil)
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("I'm a unhappy producer", errors.New("I'm not fat, I'm big-boned."))
	hc := NewHealthCheck(c, nw, p)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueProxyReachable","ok":true`, "Consumer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":true`, "Native writer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"ProducerQueueProxyReachable","ok":false`, "Producer healthcheck should be unhappy")
}

func TestHappyGTGCheckWithoutProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	hc := NewHealthCheck(c, nw, nil)

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.GTG(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")
}

func TestHappyGTGCheckWithProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("I'm a happy producer", nil)
	hc := NewHealthCheck(c, nw, p)

	req := httptest.NewRequest("GET", "http://example.com/__gtg", nil)
	w := httptest.NewRecorder()

	hc.GTG(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")
}

func TestUnhappyConsumerGTGWithoutProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm an unhappy consumer", errors.New("Screw you guys I'm going home!"))
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	hc := NewHealthCheck(c, nw, nil)

	req := httptest.NewRequest("GET", "http://example.com/__gtg", nil)
	w := httptest.NewRecorder()

	hc.GTG(w, req)

	assert.Equal(t, 503, w.Code, "It should return HTTP 503 Service Unavailable")
}

func TestUnhappyConsumerGTGWithProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm an unhappy consumer", errors.New("Screw you guys I'm going home!"))
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("I'm a happy producer", nil)
	hc := NewHealthCheck(c, nw, p)

	req := httptest.NewRequest("GET", "http://example.com/__gtg", nil)
	w := httptest.NewRecorder()

	hc.GTG(w, req)

	assert.Equal(t, 503, w.Code, "It should return HTTP 503 Service Unavailable")
}

func TestUnhappyNativeWriterGTGWithoutProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an unhappy writer", errors.New("Oh, my God, they killed Kenny!"))
	hc := NewHealthCheck(c, nw, nil)

	req := httptest.NewRequest("GET", "http://example.com/__gtg", nil)
	w := httptest.NewRecorder()

	hc.GTG(w, req)

	assert.Equal(t, 503, w.Code, "It should return HTTP 503 Service Unavailable")
}

func TestUnhappyNativeWriterGTGWithProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an unhappy writer", errors.New("Oh, my God, they killed Kenny!"))
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("I'm a happy producer", nil)
	hc := NewHealthCheck(c, nw, p)

	req := httptest.NewRequest("GET", "http://example.com/__gtg", nil)
	w := httptest.NewRecorder()

	hc.GTG(w, req)

	assert.Equal(t, 503, w.Code, "It should return HTTP 503 Service Unavailable")
}

func TestUnhappyGTGCheck(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an happy writer", nil)
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("I'm a unhappy producer", errors.New("I'm not fat, I'm big-boned."))
	hc := NewHealthCheck(c, nw, p)

	req := httptest.NewRequest("GET", "http://example.com/__gtg", nil)
	w := httptest.NewRecorder()

	hc.GTG(w, req)

	assert.Equal(t, 503, w.Code, "It should return HTTP 503 Service Unavailable")
}

type ConsumerMock struct {
	mock.Mock
}

func (c ConsumerMock) ConnectivityCheck() (string, error) {
	args := c.Called()
	return args.String(0), args.Error(1)
}

func (c ConsumerMock) Start() {
	c.Called()
}

func (c ConsumerMock) Stop() {
	c.Called()
}

type WriterMock struct {
	mock.Mock
}

func (w WriterMock) GetCollectionByOriginID(originID string) (string, error) {
	args := w.Called(originID)
	return args.String(0), args.Error(1)
}

func (w WriterMock) WriteContentBodyToCollection(ctx context.Context, cBody native.ContentBody, collection string) error {
	args := w.Called(cBody, collection)
	return args.Error(0)
}

func (w WriterMock) ConnectivityCheck() (string, error) {
	args := w.Called()
	return args.String(0), args.Error(1)
}

type ProducerMock struct {
	mock.Mock
}

func (p ProducerMock) ConnectivityCheck() (string, error) {
	args := p.Called()
	return args.String(0), args.Error(1)
}

func (p ProducerMock) SendMessage(uuid string, msg producer.Message) error {
	args := p.Called(uuid, msg)
	return args.Error(0)
}
