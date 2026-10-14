package agent_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/faust8888/go-musthave-metrics/internal/agent"
)

func TestAgent_Run_SendsMetrics(t *testing.T) {
	received := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/updates/" {
			select {
			case received <- struct{}{}:
			default:
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go agent.New(srv.URL, "", 20*time.Millisecond, 40*time.Millisecond, 1).Run(ctx)

	select {
	case <-received:
	case <-ctx.Done():
		t.Fatal("timed out waiting for agent to send metrics")
	}
}

func TestAgent_Run_RateLimit(t *testing.T) {
	const limit = 2
	var current, max atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := current.Add(1)
		for {
			m := max.Load()
			if n <= m || max.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(80 * time.Millisecond)
		current.Add(-1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go agent.New(srv.URL, "", 10*time.Millisecond, 15*time.Millisecond, limit).Run(ctx)

	time.Sleep(400 * time.Millisecond)
	cancel()

	got := max.Load()
	if got == 0 {
		t.Fatal("no requests received")
	}
	if got > limit {
		t.Errorf("max concurrent requests: got %d, want <= %d", got, limit)
	}
}
