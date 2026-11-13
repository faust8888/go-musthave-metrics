// Package agent collects runtime metrics and reports them to the server.
package agent

import (
	"context"
	"crypto/rsa"
	"log"
	"sync"
	"time"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
)

const (
	defaultHTTPTimeout  = 10 * time.Second
	defaultDrainTimeout = 30 * time.Second
)

// Agent polls runtime and gopsutil metrics in separate goroutines and sends
// snapshots through a worker pool limited by rateLimit concurrent requests.
type Agent struct {
	collector *Collector
	sender    *Sender
	poll      time.Duration
	report    time.Duration
	workers   int
	drain     time.Duration
}

// New creates an agent that polls every poll interval and reports every report
// interval using at most rateLimit concurrent HTTP workers. A non-nil pub
// encrypts outgoing bodies with the server public key.
func New(serverURL, key string, poll, report time.Duration, rateLimit int, pub *rsa.PublicKey) *Agent {
	if rateLimit < 1 {
		rateLimit = 1
	}
	return &Agent{
		collector: NewCollector(),
		sender:    NewSender(serverURL, key, pub),
		poll:      poll,
		report:    report,
		workers:   rateLimit,
		drain:     defaultDrainTimeout,
	}
}

// Run starts collectors, a reporter and a worker pool, and blocks until ctx is done.
// After a signal it still tries to deliver buffered metrics, but the wait is
// bounded by drain so a stuck HTTP call cannot hang the process.
func (a *Agent) Run(ctx context.Context) {
	jobs := make(chan []models.Metrics, a.workers)

	sendCtx, stopSend := context.WithCancel(context.Background())
	defer stopSend()

	var workers sync.WaitGroup
	for i := 0; i < a.workers; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobs {
				if err := a.sender.SendBatch(sendCtx, job); err != nil {
					log.Printf("send metrics error: %v", err)
				}
			}
		}()
	}

	var collectors sync.WaitGroup
	collectors.Add(2)
	go func() {
		defer collectors.Done()
		tick(ctx, a.poll, a.collector.Collect)
	}()
	go func() {
		defer collectors.Done()
		tick(ctx, a.poll, a.collector.CollectGopsutil)
	}()

	tick(ctx, a.report, func() {
		enqueue(ctx, jobs, a.collector.Snapshot())
	})

	drainCtx, drainCancel := context.WithTimeout(context.Background(), a.drain)
	defer drainCancel()

	waitGroup(drainCtx, &collectors)
	enqueue(drainCtx, jobs, a.collector.Snapshot())
	close(jobs)

	if !waitGroup(drainCtx, &workers) {
		stopSend()
		stopWait, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if !waitGroup(stopWait, &workers) {
			log.Printf("agent: drain timeout, dropping remaining metrics")
		}
	}
}

func enqueue(ctx context.Context, jobs chan []models.Metrics, job []models.Metrics) {
	if len(job) == 0 {
		return
	}
	select {
	case jobs <- job:
	case <-ctx.Done():
	}
}

func waitGroup(ctx context.Context, wg *sync.WaitGroup) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

func tick(ctx context.Context, d time.Duration, fn func()) {
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fn()
		}
	}
}
