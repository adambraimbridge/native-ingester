package resources

import (
	"net/http"

	fthealth "github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/native-ingester/native"
)

// HealthCheck implements the healthcheck for the native ingester
type HealthCheck struct {
	consumer consumer.MessageConsumer
	writer   native.Writer
	producer producer.MessageProducer
}

// NewHealthCheck return a new instance of a native ingester HealthCheck
func NewHealthCheck(c consumer.MessageConsumer, nw native.Writer, p producer.MessageProducer) *HealthCheck {
	return &HealthCheck{c, nw, p}
}

func (hc *HealthCheck) consumerQueueCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Native content or metadata will not reach this app, nor will they be stored in native store",
		Name:             "ConsumerQueueProxyReachable",
		PanicGuide:       "https://dewey.ft.com/native-ingester.html",
		Severity:         1,
		TechnicalSummary: "Consumer message queue proxy is not reachable/healthy",
		Checker:          hc.consumer.ConnectivityCheck,
	}
}

func (hc *HealthCheck) producerQueueCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Content will not reach the end of the publishing pipeline",
		Name:             "ProducerQueueProxyReachable",
		PanicGuide:       "https://dewey.ft.com/native-ingester.html",
		Severity:         1,
		TechnicalSummary: "Producer message queue proxy is not reachable/healthy",
		Checker:          hc.producer.ConnectivityCheck,
	}
}

func (hc *HealthCheck) nativeWriterCheck() fthealth.Check {
	return fthealth.Check{
		BusinessImpact:   "Content or metadata will not be written in the native store nor will they reach the end of the publishing pipeline",
		Name:             "NativeWriterReachable",
		PanicGuide:       "https://dewey.ft.com/native-ingester.html",
		Severity:         1,
		TechnicalSummary: "Native writer is not reachable/healthy",
		Checker:          hc.writer.ConnectivityCheck,
	}
}

//Handler returns the HTTP handler of the heatlh check
func (hc *HealthCheck) Handler() func(w http.ResponseWriter, req *http.Request) {
	checks := []fthealth.Check{hc.consumerQueueCheck(), hc.nativeWriterCheck()}
	if hc.producer != nil {
		checks = append(checks, hc.producerQueueCheck())
	}
	h := fthealth.Handler(
		"Dependent services healthcheck",
		"Checks if all the dependent services are reachable and healthy.",
		checks...,
	)
	return h
}

// GTG is the HTTP handler function for the Good-To-Go of the native ingester
func (hc *HealthCheck) GTG(w http.ResponseWriter, req *http.Request) {
	if _, err := hc.consumer.ConnectivityCheck(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	if _, err := hc.writer.ConnectivityCheck(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	if hc.producer != nil {
		if _, err := hc.producer.ConnectivityCheck(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}
}
