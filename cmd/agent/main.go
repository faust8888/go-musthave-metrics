package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/faust8888/go-musthave-metrics/internal/agent"
	"github.com/faust8888/go-musthave-metrics/internal/envconfig"
)

func main() {
	addr := flag.String("a", "localhost:8080", "HTTP server address")
	reportSec := flag.Int("r", 10, "report interval in seconds")
	pollSec := flag.Int("p", 2, "poll interval in seconds")
	flag.Parse()

	envconfig.String("ADDRESS", addr)
	if err := envconfig.Int("REPORT_INTERVAL", reportSec); err != nil {
		log.Fatal(err)
	}
	if err := envconfig.Int("POLL_INTERVAL", pollSec); err != nil {
		log.Fatal(err)
	}

	serverURL := fmt.Sprintf("http://%s", *addr)
	pollInterval := time.Duration(*pollSec) * time.Second
	reportInterval := time.Duration(*reportSec) * time.Second

	collector := agent.NewCollector()
	sender := agent.NewSender(serverURL)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	log.Printf("Agent started: server=%s poll=%ds report=%ds", serverURL, *pollSec, *reportSec)

	for {
		select {
		case <-ctx.Done():
			log.Println("Agent stopped")
			return
		case <-pollTicker.C:
			collector.Collect()
		case <-reportTicker.C:
			if err := sender.Send(collector); err != nil {
				log.Printf("send metrics error: %v", err)
			}
		}
	}
}
