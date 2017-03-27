package queue

import (
	"errors"
	"testing"

	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/native-ingester/native"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	log "github.com/Sirupsen/logrus"
	testLog "github.com/Sirupsen/logrus/hooks/test"
	"context"
)

const methodeOriginSystemID = "http://cmdb.ft.com/systems/methode-web-pub"
const methodeCollection = "methode"

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

func TestWriteToNativeSuccessfullyWithoutForward(t *testing.T) {
	w := new(WriterMock)
	w.On("GetCollectionByOriginID", methodeOriginSystemID).Return(methodeCollection, nil)
	w.On("WriteContentBodyToCollection", mock.AnythingOfType("native.ContentBody"), methodeCollection).Return(nil)

	p := new(ProducerMock)

	mh := NewMessageHandler(w)
	mh.producer = p
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertNotCalled(t, "SendMessage", mock.AnythingOfType("string"), mock.AnythingOfType("producer.Message"))
}

func TestWriteToNativeSuccessfullyWithForward(t *testing.T) {
	w := new(WriterMock)
	w.On("GetCollectionByOriginID", methodeOriginSystemID).Return(methodeCollection, nil)
	w.On("WriteContentBodyToCollection", mock.AnythingOfType("native.ContentBody"), methodeCollection).Return(nil)

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
	w.On("GetCollectionByOriginID", methodeOriginSystemID).Return(methodeCollection, nil)
	w.On("WriteContentBodyToCollection", mock.AnythingOfType("native.ContentBody"), methodeCollection).Return(nil)

	p := new(ProducerMock)

	mh := NewMessageHandler(w)
	mh.ForwardTo(p)
	mh.HandleMessage(badBodyMsg)

	w.AssertNotCalled(t, "GetCollectionByOriginID", mock.AnythingOfType("string"))
	w.AssertNotCalled(t, "WriteContentBodyToCollection", mock.AnythingOfType("native.ContentBody"), mock.AnythingOfType("string"))
	p.AssertNotCalled(t, "SendMessage", mock.AnythingOfType("string"), mock.AnythingOfType("producer.Message"))
}

func TestWriteToNativeFailWithNotCollectionForOriginId(t *testing.T) {
	w := new(WriterMock)
	w.On("GetCollectionByOriginID", methodeOriginSystemID).Return("", errors.New("Collection Not Found"))

	p := new(ProducerMock)

	mh := NewMessageHandler(w)
	mh.ForwardTo(p)
	mh.HandleMessage(goodMsg)

	w.AssertNotCalled(t, "WriteContentBodyToCollection", mock.AnythingOfType("native.ContentBody"), mock.AnythingOfType("string"))
	p.AssertNotCalled(t, "SendMessage", mock.AnythingOfType("string"), mock.AnythingOfType("producer.Message"))
}

func TestWriteToNativeFailBecauseOfWriter(t *testing.T) {
	w := new(WriterMock)
	w.On("GetCollectionByOriginID", methodeOriginSystemID).Return(methodeCollection, nil)
	w.On("WriteContentBodyToCollection", mock.AnythingOfType("native.ContentBody"), methodeCollection).Return(errors.New("I do not want to write today!"))

	p := new(ProducerMock)

	mh := NewMessageHandler(w)
	mh.ForwardTo(p)
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertNotCalled(t, "SendMessage", mock.AnythingOfType("string"), mock.AnythingOfType("producer.Message"))
}

func TestForwardFailBecauseOfProducer(t *testing.T) {
	hook := testLog.NewGlobal()
	w := new(WriterMock)
	w.On("GetCollectionByOriginID", methodeOriginSystemID).Return(methodeCollection, nil)
	w.On("WriteContentBodyToCollection", mock.AnythingOfType("native.ContentBody"), methodeCollection).Return(nil)

	p := new(ProducerMock)
	p.On("SendMessage", "", mock.AnythingOfType("producer.Message")).Return(errors.New("Today, I am not writing on a queue."))

	mh := NewMessageHandler(w)
	mh.ForwardTo(p)
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
	assert.Equal(t, log.ErrorLevel, hook.LastEntry().Level)
	assert.Equal(t, "Failed to forward consumed message to a different queue", hook.LastEntry().Message)
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
