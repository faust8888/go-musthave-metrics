package agent_test

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/faust8888/go-musthave-metrics/internal/agent"
	"github.com/faust8888/go-musthave-metrics/internal/hash"
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
	var batchReceived bool
	var counterSent bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/updates/" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		reader := r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			defer gr.Close()
			reader = io.NopCloser(gr)
		}
		var batch []struct {
			MType string `json:"type"`
		}
		if err := json.NewDecoder(reader).Decode(&batch); err == nil && len(batch) > 0 {
			batchReceived = true
			for _, m := range batch {
				if m.MType == "counter" {
					counterSent = true
				}
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := agent.NewCollector()
	c.Collect()

	s := agent.NewSender(srv.URL, "")
	if err := s.Send(c); err != nil {
		t.Fatalf("Send() returned error: %v", err)
	}

	if got := c.PollCount(); got != 0 {
		t.Errorf("PollCount after Send: got %d, want 0", got)
	}
	if !batchReceived {
		t.Error("no batch was received at /updates/")
	}
	if !counterSent {
		t.Error("no counter metric was included in the batch")
	}
}

func TestSender_Send_Retry(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			// Force a connection-level error by hijacking and closing the socket.
			hj, ok := w.(http.Hijacker)
			if !ok {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := agent.NewCollector()
	c.Collect()

	// Use zero delays so the test finishes instantly.
	s := agent.NewSenderWithDelays(srv.URL, "", []time.Duration{0, 0, 0})
	if err := s.Send(c); err != nil {
		t.Fatalf("Send() failed after retries: %v", err)
	}
	if n := attempts.Load(); n != 3 {
		t.Errorf("expected 3 attempts (1 fail + 1 fail + 1 ok), got %d", n)
	}
}

func TestSender_Send_HashHeader(t *testing.T) {
	const key = "secret"
	var gotHash string
	var raw []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHash = r.Header.Get(hash.Header)
		reader := r.Body
		if r.Header.Get("Content-Encoding") == "gzip" {
			gr, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			defer gr.Close()
			reader = io.NopCloser(gr)
		}
		var err error
		raw, err = io.ReadAll(reader)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := agent.NewCollector()
	c.Collect()

	s := agent.NewSender(srv.URL, key)
	if err := s.Send(c); err != nil {
		t.Fatalf("Send() returned error: %v", err)
	}
	if gotHash == "" {
		t.Fatal("HashSHA256 header was not set")
	}
	if !hash.Equal(gotHash, raw, key) {
		t.Errorf("HashSHA256 does not match uncompressed body")
	}
}
