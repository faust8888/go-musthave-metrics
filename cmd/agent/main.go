package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/faust8888/go-musthave-metrics/internal/agent"
)

func main() {
	addr := flag.String("a", "localhost:8080", "HTTP server address")
	reportSec := flag.Int("r", 10, "report interval in seconds")
	pollSec := flag.Int("p", 2, "poll interval in seconds")
	flag.Parse()

	if v := os.Getenv("ADDRESS"); v != "" {
		*addr = v
	}
	if v := os.Getenv("REPORT_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*reportSec = n
		}
	}
	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*pollSec = n
		}
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
