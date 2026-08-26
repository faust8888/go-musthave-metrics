package agent

import (
	"context"
	"log"
	"sync"
	"time"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
)

// Agent polls runtime and gopsutil metrics in separate goroutines and sends
// snapshots through a worker pool limited by rateLimit concurrent requests.
type Agent struct {
	collector *Collector
	sender    *Sender
	poll      time.Duration
	report    time.Duration
	workers   int
}

func New(serverURL, key string, poll, report time.Duration, rateLimit int) *Agent {
	if rateLimit < 1 {
		rateLimit = 1
	}
	return &Agent{
		collector: NewCollector(),
		sender:    NewSender(serverURL, key),
		poll:      poll,
		report:    report,
		workers:   rateLimit,
	}
}

// Run starts collectors, a reporter and a worker pool, and blocks until ctx is done.
func (a *Agent) Run(ctx context.Context) {
	jobs := make(chan []models.Metrics, a.workers)

	var workers sync.WaitGroup
	for i := 0; i < a.workers; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for job := range jobs {
				if err := a.sender.SendBatch(job); err != nil {
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
		job := a.collector.Snapshot()
		if len(job) == 0 {
			return
		}
		select {
		case jobs <- job:
		case <-ctx.Done():
		}
	})

	close(jobs)
	workers.Wait()
	collectors.Wait()
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
