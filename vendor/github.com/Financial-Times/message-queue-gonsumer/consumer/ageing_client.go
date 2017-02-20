package consumer

import (
	"log"
	"net/http"
	"time"
)

//AgeingClient defines an ageing http client for consuming messages
type AgeingClient struct {
	Client http.Client
	MaxAge time.Duration
}

//StartAgeingProcess periodically close idle connections according to the MaxAge of an AgeingClient
func (client AgeingClient) StartAgeingProcess() {
	log.Printf("INFO: Starting aging [%d]", client.MaxAge)
	ticker := time.NewTicker(client.MaxAge)
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Print("INFO: Closing idle connections")
				client.Client.Transport.(*http.Transport).CloseIdleConnections()
			}
		}
	}()
}
