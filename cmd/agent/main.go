package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/faust8888/go-musthave-metrics/internal/agent"
)

func main() {
	addr := flag.String("a", "localhost:8080", "HTTP server address")
	reportSec := flag.Int("r", 10, "report interval in seconds")
	pollSec := flag.Int("p", 2, "poll interval in seconds")
	flag.Parse()

	serverURL := fmt.Sprintf("http://%s", *addr)
	pollInterval := time.Duration(*pollSec) * time.Second
	reportInterval := time.Duration(*reportSec) * time.Second

	collector := agent.NewCollector()
	sender := agent.NewSender(serverURL)

	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	log.Printf("Agent started: server=%s poll=%ds report=%ds", serverURL, *pollSec, *reportSec)

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
