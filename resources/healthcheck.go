package resources

import (
	"net/http"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/message-queue-go-producer/producer"
	"github.com/Financial-Times/message-queue-gonsumer/consumer"
	"github.com/Financial-Times/native-ingester/native"
	"github.com/Financial-Times/service-status-go/gtg"
)

// HealthCheck implements the healthcheck for the native ingester
type HealthCheck struct {
	writer   native.Writer
	consumer consumer.MessageConsumer
	producer producer.MessageProducer
}

// NewHealthCheck return a new instance of a native ingester HealthCheck
func NewHealthCheck(c consumer.MessageConsumer, p producer.MessageProducer, nw native.Writer) *HealthCheck {
	return &HealthCheck{
		writer:   nw,
		consumer: c,
		producer: p,
	}
}

func (hc *HealthCheck) consumerQueueCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "consumer-queue-proxy",
		BusinessImpact:   "Native content or metadata will not reach this app, nor will they be stored in native store",
		Name:             "ConsumerQueueProxyReachable",
		PanicGuide:       "https://dewey.ft.com/native-ingester.html",
		Severity:         2,
		TechnicalSummary: "Consumer message queue proxy is not reachable/healthy",
		Checker:          hc.consumer.ConnectivityCheck,
	}
}

func (hc *HealthCheck) producerQueueCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "producer-queue-proxy",
		BusinessImpact:   "Content or metadata will not reach the end of the publishing pipeline",
		Name:             "ProducerQueueProxyReachable",
		PanicGuide:       "https://dewey.ft.com/native-ingester.html",
		Severity:         2,
		TechnicalSummary: "Producer message queue proxy is not reachable/healthy",
		Checker:          hc.producer.ConnectivityCheck,
	}
}

func (hc *HealthCheck) nativeWriterCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "native-writer",
		BusinessImpact:   "Content or metadata will not be written in the native store nor will they reach the end of the publishing pipeline",
		Name:             "NativeWriterReachable",
		PanicGuide:       "https://dewey.ft.com/native-ingester.html",
		Severity:         2,
		TechnicalSummary: "Native writer is not reachable/healthy",
		Checker:          hc.writer.ConnectivityCheck,
	}
}

//Handler returns the HTTP handler of the healthcheck
func (hc *HealthCheck) Handler() func(w http.ResponseWriter, req *http.Request) {
	checks := []fthealth.Check{hc.consumerQueueCheck(), hc.nativeWriterCheck()}
	if hc.producer != nil {
		checks = append(checks, hc.producerQueueCheck())
	}

	healthCheck := fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  "native-ingester",
			Name:        "Native Ingester Healthcheck",
			Description: "It checks if kafka proxy and native writer are available",
			Checks:      checks,
		},
		Timeout: 10 * time.Second,
	}

	return fthealth.Handler(healthCheck)
}

func (hc *HealthCheck) GTG() gtg.Status {
	consumerCheck := func() gtg.Status {
		return gtgCheck(hc.consumer.ConnectivityCheck)
	}

	writerCheck := func() gtg.Status {
		return gtgCheck(hc.writer.ConnectivityCheck)
	}

	if hc.producer != nil {
		producerCheck := func() gtg.Status {
			return gtgCheck(hc.producer.ConnectivityCheck)
		}
		return gtg.FailFastParallelCheck([]gtg.StatusChecker{consumerCheck, producerCheck, writerCheck})()
	}

	return gtg.FailFastParallelCheck([]gtg.StatusChecker{consumerCheck, writerCheck})()
}

func gtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}
