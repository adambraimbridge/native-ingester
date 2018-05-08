package mocks

import (
	"github.com/Financial-Times/kafka-client-go/kafka"
	"github.com/Financial-Times/native-ingester/native"
	"github.com/stretchr/testify/mock"
)

type ProducerMock struct {
	mock.Mock
}

func (p *ProducerMock) ConnectivityCheck() error {
	args := p.Called()
	return args.Error(0)
}

func (p *ProducerMock) SendMessage(msg kafka.FTMessage) error {
	args := p.Called(msg)
	return args.Error(0)
}

func (p *ProducerMock) 	Shutdown() {
	p.Called()
}

type ConsumerMock struct {
	mock.Mock
}

func (c *ConsumerMock) ConnectivityCheck() error {
	args := c.Called()
	return args.Error(0)
}

func (c *ConsumerMock) StartListening(messageHandler func(message kafka.FTMessage) error) {
	c.Called(messageHandler)
}

func (c *ConsumerMock) Shutdown() {
	c.Called()
}

type WriterMock struct {
	mock.Mock
}

func (w *WriterMock) GetCollectionByOriginID(originID string) (string, error) {
	args := w.Called(originID)
	return args.String(0), args.Error(1)
}

func (w *WriterMock) WriteToCollection(msg native.NativeMessage, collection string) (string, error) {
	args := w.Called(msg, collection)
	return args.String(0), args.Error(1)
}

func (w *WriterMock) ConnectivityCheck() (string, error) {
	args := w.Called()
	return args.String(0), args.Error(1)
}
