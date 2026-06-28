package main

import (
	"log"
	"time"

	"github.com/faust8888/go-musthave-metrics/internal/agent"
)

const (
	pollInterval   = 2 * time.Second
	reportInterval = 10 * time.Second
	serverURL      = "http://localhost:8080"
)

func main() {
	collector := agent.NewCollector()
	sender := agent.NewSender(serverURL)

	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	for {
		select {
		case <-pollTicker.C:
			collector.Collect()
		case <-reportTicker.C:
			if err := sender.Send(collector); err != nil {
				log.Printf("send metrics error: %v", err)
			}
		}
	}
}
