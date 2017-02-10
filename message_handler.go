package main

import (
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/native-ingester/native"
)

type messageHandler struct {
	writer native.Writer
}

func newMessageHandler(writer native.Writer) *messageHandler {
	return &messageHandler{writer}
}

//TODO implement forward
func (mh *messageHandler) handleMessage(msg consumer.Message) {
	mh.writer.Write(msg)
}
