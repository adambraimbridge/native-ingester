package consumer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type queueCaller interface {
	createConsumerInstance() (consumer, error)
	consumeMessages(c consumer) ([]Message, error)
	destroyConsumerInstance(c consumer) error
	commitOffsets(c consumer) error
	checkConnectivity() error
}

type defaultQueueCaller struct {
	//pool of queue addresses
	//the active address is changed in a round-robin fashion before each new consumer instance creation
	addrs []string
	//used queue addr
	//this gets 'incremented modulo' at each createConsumerInstance() call
	addrInd          int
	group            string
	topic            string
	offset           string
	caller           httpCaller
	autoCommitEnable bool
}

type httpCaller interface {
	DoReq(method, addr string, body io.Reader, headers map[string]string, expectedStatus int) ([]byte, error)
}

func (q *defaultQueueCaller) createConsumerInstance() (c consumer, err error) {
	q.addrInd = (q.addrInd + 1) % len(q.addrs)
	addr := q.addrs[q.addrInd]

	createConsumerReq := `{"auto.offset.reset": "` + q.offset + `", "auto.commit.enable": "` + strconv.FormatBool(q.autoCommitEnable) + `"}`
	data, err := q.caller.DoReq("POST", addr+"/consumers/"+q.group, strings.NewReader(createConsumerReq), map[string]string{"Content-Type": "application/json"}, http.StatusOK)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &c)
	if err != nil {
		log.Printf("ERROR - unmarshalling json content: %s", err.Error())
		return
	}
	return
}

func (q *defaultQueueCaller) destroyConsumerInstance(c consumer) (err error) {
	url, _ := q.buildConsumerURL(c)
	_, err = q.caller.DoReq("DELETE", url.String(), nil, nil, http.StatusNoContent)
	return
}

func (q *defaultQueueCaller) consumeMessages(c consumer) ([]Message, error) {
	uri, _ := q.buildConsumerURL(c)
	uri.Path = strings.TrimRight(uri.Path, "/") + "/topics/" + q.topic
	data, err := q.caller.DoReq("GET", uri.String(), nil, map[string]string{"Accept": "application/json"}, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return parseResponse(data)
}

func (q *defaultQueueCaller) commitOffsets(c consumer) (err error) {
	url, _ := q.buildConsumerURL(c)
	url.Path = strings.TrimRight(url.Path, "/") + "/offsets"
	_, err = q.caller.DoReq("POST", url.String(), nil, nil, http.StatusOK)
	return
}

func (q *defaultQueueCaller) buildConsumerURL(c consumer) (uri *url.URL, err error) {
	uri, err = url.Parse(c.BaseURI)
	if err != nil {
		log.Printf("ERROR - parsing base URI: %s", err.Error())
		return
	}
	addr := q.addrs[q.addrInd]
	addrURL, err := url.Parse(addr)
	if err != nil {
		log.Printf("ERROR - parsing Addr: %s", err.Error())
	}
	addrURL.Path = addrURL.Path + uri.Path
	return addrURL, err
}

type defaultHTTPCaller struct {
	hostHeader       string
	authorizationKey string
	client           http.Client
}

func (caller defaultHTTPCaller) DoReq(method, url string, body io.Reader, headers map[string]string, expectedStatus int) (data []byte, err error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Printf("ERROR - creating request: %s", err.Error())
		return
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}
	if len(caller.hostHeader) > 0 {
		req.Host = caller.hostHeader
	}

	if len(caller.authorizationKey) > 0 {
		req.Header.Add("Authorization", caller.authorizationKey)
	}

	resp, err := caller.client.Do(req)
	if err != nil {
		log.Printf("ERROR - executing request: %s", err.Error())
		return
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 500 {
			// This might be a problem with the server instance, which may have been taken out
			// of the DNS pool, but because we might still have a tcp connection open, we'll
			// never re-do the DNS lookup and get a connection to a working server.  So when we
			// get 5xx, close idle connections to force the next requests to re-connect.
			if t, ok := caller.client.Transport.(*http.Transport); ok {
				t.CloseIdleConnections()
			}
		}
	}()

	if resp.StatusCode != expectedStatus {
		err = fmt.Errorf("Unexpected response status %d. Expected: %d", resp.StatusCode, expectedStatus)
		log.Printf("ERROR - %s", err.Error())
		return
	}

	return ioutil.ReadAll(resp.Body)
}

func (q *defaultQueueCaller) checkConnectivity() error {
	errMsg := ""
	for _, address := range q.addrs {
		if err := q.checkMessageQueueProxyReachable(address); err != nil {
			errMsg = errMsg + err.Error() + "; "
		}
	}
	if errMsg != "" {
		return errors.New(errMsg)
	}
	return nil
}

func (q *defaultQueueCaller) checkMessageQueueProxyReachable(address string) error {
	body, err := q.caller.DoReq("GET", address+"/topics", nil, map[string]string{"Accept": "application/json"}, http.StatusOK)
	if err != nil {
		return errors.New("Could not connect to proxy: " + err.Error())
	}
	return checkIfTopicIsPresent(body, q.topic)
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
