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
	key := flag.String("k", "", "hash key")
	rateLimit := flag.Int("l", 1, "max concurrent outgoing requests")
	flag.Parse()

	envconfig.String("ADDRESS", addr)
	envconfig.String("KEY", key)
	if err := envconfig.Int("REPORT_INTERVAL", reportSec); err != nil {
		log.Fatal(err)
	}
	if err := envconfig.Int("POLL_INTERVAL", pollSec); err != nil {
		log.Fatal(err)
	}
	if err := envconfig.Int("RATE_LIMIT", rateLimit); err != nil {
		log.Fatal(err)
	}

	serverURL := fmt.Sprintf("http://%s", *addr)
	pollInterval := time.Duration(*pollSec) * time.Second
	reportInterval := time.Duration(*reportSec) * time.Second

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("Agent started: server=%s poll=%ds report=%ds rate_limit=%d",
		serverURL, *pollSec, *reportSec, *rateLimit)

	agent.New(serverURL, *key, pollInterval, reportInterval, *rateLimit).Run(ctx)
	log.Println("Agent stopped")
}
