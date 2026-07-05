package agent_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/faust8888/go-musthave-metrics/internal/agent"
)

func TestCollector_Collect_IncrementsPollCount(t *testing.T) {
	c := agent.NewCollector()

	c.Collect()
	if got := c.PollCount(); got != 1 {
		t.Errorf("PollCount after 1 collect: got %d, want 1", got)
	}

	c.Collect()
	if got := c.PollCount(); got != 2 {
		t.Errorf("PollCount after 2 collects: got %d, want 2", got)
	}
}

func TestCollector_Collect_PopulatesGauges(t *testing.T) {
	c := agent.NewCollector()
	c.Collect()

	gauges := c.Gauges()
	required := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "RandomValue",
	}
	for _, name := range required {
		if _, ok := gauges[name]; !ok {
			t.Errorf("gauge %q not found after Collect()", name)
		}
	}
}

func TestCollector_TakeAndResetPollCount(t *testing.T) {
	c := agent.NewCollector()
	c.Collect()
	c.Collect()

	got := c.TakeAndResetPollCount()
	if got != 2 {
		t.Errorf("TakeAndResetPollCount: got %d, want 2", got)
	}
	if after := c.PollCount(); after != 0 {
		t.Errorf("PollCount after take-reset: got %d, want 0", after)
	}
}

func TestSender_Send(t *testing.T) {
	received := make(map[string]int)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		received[r.URL.Path]++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := agent.NewCollector()
	c.Collect()

	s := agent.NewSender(srv.URL)
	if err := s.Send(c); err != nil {
		t.Fatalf("Send() returned error: %v", err)
	}

	// PollCount should be reset after send
	if got := c.PollCount(); got != 0 {
		t.Errorf("PollCount after Send: got %d, want 0", got)
	}

	// Counter metric must have been sent
	found := false
	for path := range received {
		if len(path) > 16 && path[:16] == "/update/counter/" {
			found = true
			break
		}
	}
	if !found {
		t.Error("no counter metric was sent to server")
	}
}
