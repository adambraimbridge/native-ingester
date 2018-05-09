package resources

import (
	"net/http"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/kafka-client-go/kafka"
	"github.com/Financial-Times/native-ingester/native"
	"github.com/Financial-Times/service-status-go/gtg"
)

// HealthCheck implements the healthcheck for the native ingester
type HealthCheck struct {
	writer   native.Writer
	consumer kafka.Consumer
	producer kafka.Producer
}

// NewHealthCheck return a new instance of a native ingester HealthCheck
func NewHealthCheck(c kafka.Consumer, p kafka.Producer, nw native.Writer) *HealthCheck {
	return &HealthCheck{
		writer:   nw,
		consumer: c,
		producer: p,
	}
}

func (hc *HealthCheck) consumerQueueCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "consumer-queue",
		BusinessImpact:   "Native content or metadata will not reach this app, nor will they be stored in native store",
		Name:             "ConsumerQueueReachable",
		PanicGuide:       "https://dewey.ft.com/native-ingester.html",
		Severity:         2,
		TechnicalSummary: "Consumer message queue is not reachable/healthy",
		Checker:          check(hc.consumer.ConnectivityCheck),
	}
}

func (hc *HealthCheck) producerQueueCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "producer-queue",
		BusinessImpact:   "Content or metadata will not reach the end of the publishing pipeline",
		Name:             "ProducerQueueReachable",
		PanicGuide:       "https://dewey.ft.com/native-ingester.html",
		Severity:         2,
		TechnicalSummary: "Producer message queue is not reachable/healthy",
		Checker:          check(hc.producer.ConnectivityCheck),
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

func check(fn func() error) func() (string,error) {
	return func() (string,error) {
		msg := "OK"
		err := fn();
		if err != nil {
			msg = err.Error();
		}

		return msg, err
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
			Description: "It checks if kafka and native writer are available",
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
		return writerGtgCheck(hc.writer.ConnectivityCheck)
	}

	if hc.producer != nil {
		producerCheck := func() gtg.Status {
			return gtgCheck(hc.producer.ConnectivityCheck)
		}
		return gtg.FailFastParallelCheck([]gtg.StatusChecker{consumerCheck, producerCheck, writerCheck})()
	}

	return gtg.FailFastParallelCheck([]gtg.StatusChecker{consumerCheck, writerCheck})()
}

func gtgCheck(handler func() error) gtg.Status {
	if err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}

func writerGtgCheck(handler func() (string,error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}
