package resources

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Financial-Times/native-ingester/native"
)

func TestNewHealthCheckWithoutProducerConfig(t *testing.T) {
	c := &consumer.QueueConfig{}
	nw := new(WriterMock)
	hc := NewHealthCheck(c, nil, nw)

	assert.Nil(t, hc.producer)
	assert.NotNil(t, hc.consumer)
	assert.NotNil(t, hc.writer)
}

func TestNewHealthCheckWithProducerConfig(t *testing.T) {
	c := &consumer.QueueConfig{}
	p := &producer.MessageProducerConfig{}
	nw := new(WriterMock)
	hc := NewHealthCheck(c, p, nw)

	assert.NotNil(t, hc.producer)
	assert.NotNil(t, hc.consumer)
	assert.NotNil(t, hc.writer)
}

func TestHappyHealthCheckWithoutProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
	}

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
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
	}

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
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
	}

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
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
	}

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
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
	}

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
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
	}

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
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
	}

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
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
	}

	status := hc.GTG()

	assert.True(t, status.GoodToGo)
	assert.Empty(t, status.Message)
}

func TestHappyGTGCheckWithProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("I'm a happy producer", nil)
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
	}

	status := hc.GTG()

	assert.True(t, status.GoodToGo)
	assert.Empty(t, status.Message)
}

func TestUnhappyConsumerGTGWithoutProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm an unhappy consumer", errors.New("Screw you guys I'm going home!"))
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
	}

	status := hc.GTG()

	assert.False(t, status.GoodToGo)
	assert.Equal(t, "Screw you guys I'm going home!", status.Message)
}

func TestUnhappyConsumerGTGWithProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm an unhappy consumer", errors.New("Screw you guys I'm going home!"))
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("I'm a happy producer", nil)
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
	}

	status := hc.GTG()

	assert.False(t, status.GoodToGo)
	assert.Equal(t, "Screw you guys I'm going home!", status.Message)
}

func TestUnhappyNativeWriterGTGWithoutProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an unhappy writer", errors.New("Oh, my God, they killed Kenny!"))
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
	}

	status := hc.GTG()

	assert.False(t, status.GoodToGo)
	assert.Equal(t, "Oh, my God, they killed Kenny!", status.Message)
}

func TestUnhappyNativeWriterGTGWithProducer(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an unhappy writer", errors.New("Oh, my God, they killed Kenny!"))
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("I'm a happy producer", nil)
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
	}

	status := hc.GTG()

	assert.False(t, status.GoodToGo)
	assert.Equal(t, "Oh, my God, they killed Kenny!", status.Message)
}

func TestUnhappyGTGCheck(t *testing.T) {
	c := new(ConsumerMock)
	c.On("ConnectivityCheck").Return("I'm a happy consumer", nil)
	nw := new(WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an happy writer", nil)
	p := new(ProducerMock)
	p.On("ConnectivityCheck").Return("I'm a unhappy producer", errors.New("I'm not fat, I'm big-boned."))
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
	}

	status := hc.GTG()

	assert.False(t, status.GoodToGo)
	assert.Equal(t, "I'm not fat, I'm big-boned.", status.Message)
}

type ConsumerMock struct {
	mock.Mock
}

func (c *ConsumerMock) ConnectivityCheck() (string, error) {
	args := c.Called()
	return args.String(0), args.Error(1)
}

func (c *ConsumerMock) Start() {
	c.Called()
}

func (c *ConsumerMock) Stop() {
	c.Called()
}

type WriterMock struct {
	mock.Mock
}

func (w *WriterMock) GetCollectionByOriginID(originID string) (string, error) {
	args := w.Called(originID)
	return args.String(0), args.Error(1)
}

func (w *WriterMock) WriteToCollection(msg native.NativeMessage, collection string) error {
	args := w.Called(msg, collection)
	return args.Error(0)
}

func (w *WriterMock) ConnectivityCheck() (string, error) {
	args := w.Called()
	return args.String(0), args.Error(1)
}

type ProducerMock struct {
	mock.Mock
}

func (p *ProducerMock) ConnectivityCheck() (string, error) {
	args := p.Called()
	return args.String(0), args.Error(1)
}

func (p *ProducerMock) SendMessage(uuid string, msg producer.Message) error {
	args := p.Called(uuid, msg)
	return args.Error(0)
}
