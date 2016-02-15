package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Financial-Times/go-fthealth"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
)

// Healthcheck offers methods to measure application health.
type Healthcheck struct {
	client           http.Client
	srcConf          consumer.QueueConfig
	nativeWriterConf NativeWriterConfig
}

func (h *Healthcheck) checkHealth() func(w http.ResponseWriter, r *http.Request) {
	return fthealth.HandlerParallel("Dependent services healthcheck", "Checks if all the dependent services are reachable and healthy.", h.messageQueueProxyReachable(), h.nativeWriterReachable())
}

func (h *Healthcheck) gtg(writer http.ResponseWriter, req *http.Request) {
	healthChecks := []func() error{h.checkAggregateMessageQueueProxiesReachable, h.checkNativeWriterHealthy}

	for _, hCheck := range healthChecks {
		if err := hCheck(); err != nil {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}
}

func (h *Healthcheck) messageQueueProxyReachable() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Content V1 Metadata is not suggested. This will negatively impact V1 metadata availability.",
		Name:             "MessageQueueProxyReachable",
		PanicGuide:       "https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/extra-publishing/v1-suggestor-runbook",
		Severity:         1,
		TechnicalSummary: "Message queue proxy is not reachable/healthy",
		Checker:          h.checkAggregateMessageQueueProxiesReachable,
	}

}
func (h *Healthcheck) nativeWriterReachable() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Low impact. Native content is not persisted.",
		Name:             "NativeWriterReachable",
		PanicGuide:       "https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/extra-publishing/v1-suggestor-runbook",
		Severity:         1,
		TechnicalSummary: "Native writer is not reachable/healthy",
		Checker:          h.checkNativeWriterHealthy,
	}

}

func (h *Healthcheck) checkAggregateMessageQueueProxiesReachable() error {
	errMsg := ""
	for i := 0; i < len(h.srcConf.Addrs); i++ {
		err := h.checkMessageQueueProxyReachable(h.srcConf.Addrs[i], h.srcConf.Topic, h.srcConf.AuthorizationKey, h.srcConf.Queue)
		if err == nil {
			return nil
		}
		errMsg = errMsg + fmt.Sprintf("For %s there is an error %v \n", h.srcConf.Addrs[i], err.Error())
	}
	return errors.New(errMsg)
}

func (h *Healthcheck) checkMessageQueueProxyReachable(address string, topic string, authKey string, queue string) error {
	req, err := http.NewRequest("GET", address+"/topics", nil)
	if err != nil {
		warnLogger.Printf("Could not connect to proxy: %v", err.Error())
		return err
	}

	if len(authKey) > 0 {
		req.Header.Add("Authorization", authKey)
	}

	if len(queue) > 0 {
		req.Host = queue
	}

	resp, err := h.client.Do(req)
	if err != nil {
		warnLogger.Printf("Could not connect to proxy: %v", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Proxy returned status: %d", resp.StatusCode)
		return errors.New(errMsg)
	}

	body, err := ioutil.ReadAll(resp.Body)
	return checkIfTopicIsPresent(body, topic)

}

func (h *Healthcheck) checkNativeWriterHealthy() error {
	address := h.nativeWriterConf.Address
	req, err := http.NewRequest("GET", address+"/__gtg", nil)
	if err != nil {
		warnLogger.Printf("Could not create request to native writer at [%s]: [%v]", address, err.Error())
		return err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		warnLogger.Printf("Could not connect to native writer at [%s]: [%v]", address, err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Native writer returned: [%d]", resp.StatusCode)
		return errors.New(errMsg)
	}

	ioutil.ReadAll(resp.Body)
	return nil
}

func checkIfTopicIsPresent(body []byte, searchedTopic string) error {
	var topics []string

	err := json.Unmarshal(body, &topics)
	if err != nil {
		return fmt.Errorf("Error occured and topic could not be found. %v", err.Error())
	}

	for _, topic := range topics {
		if topic == searchedTopic {
			return nil
		}
	}

	return errors.New("Topic was not found")
}
