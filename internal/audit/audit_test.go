package audit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type recordingSink struct {
	mu     sync.Mutex
	events []Event
}

func (s *recordingSink) Write(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
}

func (s *recordingSink) snapshot() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Event, len(s.events))
	copy(out, s.events)
	return out
}

func newTestAuditor(t *testing.T) *Auditor {
	t.Helper()
	a, err := New("", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return a
}

func TestAuditor_NotifyAllSinks(t *testing.T) {
	a := newTestAuditor(t)
	first := &recordingSink{}
	second := &recordingSink{}
	a.Add(first)
	a.Add(second)

	event := Event{TS: 1, Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"}
	a.Notify(event)

	if len(first.events) != 1 || len(second.events) != 1 {
		t.Fatalf("want 1 event each, got a=%d b=%d", len(first.events), len(second.events))
	}
	if first.events[0].Metrics[0] != "Alloc" || second.events[0].IPAddress != "127.0.0.1" {
		t.Errorf("sinks got unexpected event: %+v %+v", first.events[0], second.events[0])
	}
}

func TestAuditor_RecordNilSafe(t *testing.T) {
	var a *Auditor
	req := httptest.NewRequest(http.MethodPost, "/update", nil)
	a.Record(req, []string{"Alloc"})
	_ = a.Close()
}

func TestFileSink_AppendsJSONLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	sink, err := NewFileSink(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sink.Close() })

	sink.Write(Event{TS: 42, Metrics: []string{"Alloc", "Frees"}, IPAddress: "192.168.0.42"})
	sink.Write(Event{TS: 43, Metrics: []string{"HeapSys"}, IPAddress: "10.0.0.1"})

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var events []Event
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e Event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatalf("invalid json line %q: %v", sc.Text(), err)
		}
		events = append(events, e)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("want 2 lines, got %d", len(events))
	}
	if events[0].TS != 42 || events[0].IPAddress != "192.168.0.42" {
		t.Errorf("first event: %+v", events[0])
	}
	if got, want := events[0].Metrics, []string{"Alloc", "Frees"}; len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("metrics: got %v, want %v", got, want)
	}
}

func TestFileSink_OpenFailsOnBadPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "audit.log")
	_, err := NewFileSink(path)
	if err == nil {
		t.Fatal("want error for missing parent directory")
	}
	_, err = New(path, "")
	if err == nil {
		t.Fatal("New should fail when the audit file cannot be opened")
	}
}

func TestHTTPSink_PostsJSON(t *testing.T) {
	var got Event
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type: got %q, want application/json", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	NewHTTPSink(srv.URL).Write(Event{
		TS:        99,
		Metrics:   []string{"PollCount"},
		IPAddress: "127.0.0.1",
	})

	if got.TS != 99 || got.IPAddress != "127.0.0.1" || len(got.Metrics) != 1 || got.Metrics[0] != "PollCount" {
		t.Errorf("posted event: %+v", got)
	}
}

func TestHTTPSink_LogsNonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })

	NewHTTPSink(srv.URL).Write(Event{TS: 1, Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"})

	if !strings.Contains(buf.String(), "audit http status: 500") {
		t.Errorf("want status log, got %q", buf.String())
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "192.168.0.42:5555"
	if got := clientIP(req); got != "192.168.0.42" {
		t.Errorf("RemoteAddr: got %q", got)
	}

	req.Header.Set("X-Forwarded-For", "10.1.2.3, 10.0.0.1")
	if got := clientIP(req); got != "10.1.2.3" {
		t.Errorf("X-Forwarded-For: got %q", got)
	}

	req.Header.Set("X-Real-IP", "8.8.8.8")
	if got := clientIP(req); got != "8.8.8.8" {
		t.Errorf("X-Real-IP: got %q", got)
	}
}

func TestRecord_BuildsEvent(t *testing.T) {
	sink := &recordingSink{}
	a := newTestAuditor(t)
	a.Add(sink)

	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	req.RemoteAddr = "192.168.0.42:9"
	before := time.Now().Unix()
	a.Record(req, []string{"Alloc", "Frees"})
	a.Flush()
	after := time.Now().Unix()

	events := sink.snapshot()
	if len(events) != 1 {
		t.Fatalf("want 1 event, got %d", len(events))
	}
	e := events[0]
	if e.IPAddress != "192.168.0.42" {
		t.Errorf("ip: got %q", e.IPAddress)
	}
	if len(e.Metrics) != 2 || e.Metrics[0] != "Alloc" || e.Metrics[1] != "Frees" {
		t.Errorf("metrics: %v", e.Metrics)
	}
	if e.TS < before || e.TS > after {
		t.Errorf("ts %d not in [%d, %d]", e.TS, before, after)
	}
}
