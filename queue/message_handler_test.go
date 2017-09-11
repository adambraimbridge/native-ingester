package queue

import (
	"errors"
	"github.com/Financial-Times/go-logger"
	"testing"

	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/native-ingester/native"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	methodeOriginSystemID = "http://cmdb.ft.com/systems/methode-web-pub"
	methodeCollection     = "methode"
)

var goodMsgHeaders = map[string]string{
	"X-Request-Id":      "tid_test",
	"Message-Timestamp": "2017-02-16T12:56:16Z",
	"Origin-System-Id":  methodeOriginSystemID,
}

var goodMsg = consumer.Message{
	Body:    "{}",
	Headers: goodMsgHeaders,
}

var badBodyMsg = consumer.Message{
	Body:    "I am not JSON",
	Headers: goodMsgHeaders,
}

func init() {
	logger.InitDefaultLogger("native-ingester")
}

func TestWriteToNativeSuccessfullyWithoutForward(t *testing.T) {
	w := new(WriterMock)
	w.On("GetCollectionByOriginID", methodeOriginSystemID).Return(methodeCollection, nil)
	w.On("WriteToCollection", mock.AnythingOfType("native.NativeMessage"), methodeCollection).Return("", nil)

	p := new(ProducerMock)

	mh := NewMessageHandler(w)
	mh.producer = p
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
}

func TestWriteToNativeSuccessfullyWithForward(t *testing.T) {
	w := new(WriterMock)
	w.On("GetCollectionByOriginID", methodeOriginSystemID).Return(methodeCollection, nil)
	w.On("WriteToCollection", mock.AnythingOfType("native.NativeMessage"), methodeCollection).Return("", nil)

	p := new(ProducerMock)
	p.On("SendMessage", "", mock.AnythingOfType("producer.Message")).Return(nil)

	mh := NewMessageHandler(w)
	mh.ForwardTo(p)
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
}

func TestWriteToNativeFailWithBadBodyMessage(t *testing.T) {
	w := new(WriterMock)

	p := new(ProducerMock)

	mh := NewMessageHandler(w)
	mh.ForwardTo(p)
	mh.HandleMessage(badBodyMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
}

func TestWriteToNativeFailWithNotCollectionForOriginId(t *testing.T) {
	w := new(WriterMock)
	w.On("GetCollectionByOriginID", methodeOriginSystemID).Return("", errors.New("Collection Not Found"))

	p := new(ProducerMock)

	mh := NewMessageHandler(w)
	mh.ForwardTo(p)
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
}

func TestWriteToNativeFailBecauseOfWriter(t *testing.T) {
	w := new(WriterMock)
	w.On("GetCollectionByOriginID", methodeOriginSystemID).Return(methodeCollection, nil)
	w.On("WriteToCollection", mock.AnythingOfType("native.NativeMessage"), methodeCollection).Return("", errors.New("I do not want to write today!"))

	p := new(ProducerMock)

	mh := NewMessageHandler(w)
	mh.ForwardTo(p)
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
}

func TestForwardFailBecauseOfProducer(t *testing.T) {
	hook := logger.NewTestHook("native-ingester")
	w := new(WriterMock)
	w.On("GetCollectionByOriginID", methodeOriginSystemID).Return(methodeCollection, nil)
	w.On("WriteToCollection", mock.AnythingOfType("native.NativeMessage"), methodeCollection).Return("", nil)

	p := new(ProducerMock)
	p.On("SendMessage", "", mock.AnythingOfType("producer.Message")).Return(errors.New("Today, I am not writing on a queue."))

	mh := NewMessageHandler(w)
	mh.ForwardTo(p)
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
	assert.Equal(t, "error", hook.LastEntry().Level.String())
	assert.Equal(t, "Failed to forward consumed message to a different queue", hook.LastEntry().Message)
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
