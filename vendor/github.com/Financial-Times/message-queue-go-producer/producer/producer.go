package producer

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

const contentTypeHeader = "application/vnd.kafka.binary.v1+json"
const crlf = "\r\n"

// MessageProducer defines the interface for message producer - which writes to kafka through the proxy
type MessageProducer interface {
	SendMessage(string, Message) error
	ConnectivityCheck() (string, error)
}

// DefaultMessageProducer defines default implementation of a message producer
type DefaultMessageProducer struct {
	config MessageProducerConfig
	client *http.Client
}

// MessageProducerConfig specifies the configuration for message producer
type MessageProducerConfig struct {
	//proxy address
	Addr  string `json:"address"`
	Topic string `json:"topic"`
	//the name of the queue
	//leave it empty for requests to UCS kafka-proxy
	Queue         string `json:"queue"`
	Authorization string `json:"authorization"`
}

// Message is the higher-level representation of messages from the queue: containing headers and message body
type Message struct {
	Headers map[string]string
	Body    string
}

// MessageWithRecords is a message format required by Kafka-Proxy containing all the Messages
type MessageWithRecords struct {
	Records []MessageRecord `json:"records"`
}

// MessageRecord is a Message format required by Kafka-Proxy
type MessageRecord struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NewMessageProducer returns a producer instance
func NewMessageProducer(config MessageProducerConfig) MessageProducer {
	return NewMessageProducerWithHTTPClient(config, &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		}})
}

// NewMessageProducerWithHTTPClient returns a producer instance with specified http client instance
func NewMessageProducerWithHTTPClient(config MessageProducerConfig, httpClient *http.Client) MessageProducer {
	return &DefaultMessageProducer{config, httpClient}
}

// SendMessage is the producer method that takes care of sending a message on the queue
func (p *DefaultMessageProducer) SendMessage(uuid string, message Message) (err error) {

	//concatenate message headers with message body to form a proper message string to save
	messageString := buildMessage(message)
	return p.SendRawMessage(uuid, messageString)
}

// SendRawMessage is the producer method that takes care of sending a raw message on the queue
func (p *DefaultMessageProducer) SendRawMessage(uuid string, message string) (err error) {

	//encode in base64 and envelope the message
	envelopedMessage, err := envelopeMessage(uuid, message)
	if err != nil {
		return
	}

	//create request
	req, err := constructRequest(p.config.Addr, p.config.Topic, p.config.Queue, p.config.Authorization, envelopedMessage)

	//make request
	resp, err := p.client.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("ERROR - executing request: %s", err.Error())
		return errors.New(errMsg)
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	//send - verify response status
	//log if error happens
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("ERROR - Unexpected response status %d. Expected: %d. %s", resp.StatusCode, http.StatusOK, resp.Request.URL.String())
		return errors.New(errMsg)
	}

	return nil
}

func constructRequest(addr string, topic string, queue string, authorizationKey string, message string) (*http.Request, error) {

	req, err := http.NewRequest("POST", addr+"/topics/"+topic, strings.NewReader(message))
	if err != nil {
		errMsg := fmt.Sprintf("ERROR - creating request: %s", err.Error())
		return req, errors.New(errMsg)
	}

	//set content-type header to json, and host header according to vulcand routing strategy
	req.Header.Add("Content-Type", contentTypeHeader)
	if len(queue) > 0 {
		req.Host = queue
	}

	if len(authorizationKey) > 0 {
		req.Header.Add("Authorization", authorizationKey)
	}

	return req, err
}

func buildMessage(message Message) string {

	builtMessage := "FTMSG/1.0" + crlf

	var keys []string

	//order headers
	for header := range message.Headers {
		keys = append(keys, header)
	}
	sort.Strings(keys)

	//set headers
	for _, key := range keys {
		builtMessage = builtMessage + key + ": " + message.Headers[key] + crlf
	}

	builtMessage = builtMessage + crlf + message.Body

	return builtMessage

}

func envelopeMessage(key string, message string) (string, error) {

	var key64 string
	if key != "" {
		key64 = base64.StdEncoding.EncodeToString([]byte(key))
	}

	message64 := base64.StdEncoding.EncodeToString([]byte(message))

	record := MessageRecord{Key: key64, Value: message64}
	msgWithRecords := &MessageWithRecords{Records: []MessageRecord{record}}

	jsonRecords, err := json.Marshal(msgWithRecords)

	if err != nil {
		errMsg := fmt.Sprintf("ERROR - marshalling in json: %s", err.Error())
		return "", errors.New(errMsg)
	}

	return string(jsonRecords), err
}

// ConnectivityCheck verifies if the kakfa proxy is availabe
func (p *DefaultMessageProducer) ConnectivityCheck() (string, error) {
	err := p.checkMessageQueueProxyReachable()
	if err == nil {
		return "Connectivity to producer proxy is OK.", nil
	}
	log.Printf("ERROR - Producer Connectivity Check - %s", err)
	return "Error connecting to producer proxy", err
}

func (p *DefaultMessageProducer) checkMessageQueueProxyReachable() error {
	req, err := http.NewRequest("GET", p.config.Addr+"/topics", nil)
	if err != nil {
		return fmt.Errorf("Could not connect to proxy: %v", err.Error())
	}

	if len(p.config.Authorization) > 0 {
		req.Header.Add("Authorization", p.config.Authorization)
	}

	if len(p.config.Queue) > 0 {
		req.Host = p.config.Queue
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("Could not connect to proxy: %v", err.Error())
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Producer proxy returned status: %d", resp.StatusCode)
		return errors.New(errMsg)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	return checkIfTopicIsPresent(body, p.config.Topic)
}

func checkIfTopicIsPresent(body []byte, searchedTopic string) error {
	var topics []string

	err := json.Unmarshal(body, &topics)
	if err != nil {
		return fmt.Errorf("Error occurred and topic could not be found. %v", err.Error())
	}

	for _, topic := range topics {
		if topic == searchedTopic {
			return nil
		}
	}

	return errors.New("Topic was not found")
}
