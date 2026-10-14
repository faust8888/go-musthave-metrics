package audit

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type recordingObserver struct {
	mu     sync.Mutex
	events []Event
}

func (o *recordingObserver) Update(e Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, e)
}

func TestSubject_NotifyAllObservers(t *testing.T) {
	s := New("", "")
	a := &recordingObserver{}
	b := &recordingObserver{}
	s.Attach(a)
	s.Attach(b)

	event := Event{TS: 1, Metrics: []string{"Alloc"}, IPAddress: "127.0.0.1"}
	s.Notify(event)

	if len(a.events) != 1 || len(b.events) != 1 {
		t.Fatalf("want 1 event each, got a=%d b=%d", len(a.events), len(b.events))
	}
	if a.events[0].Metrics[0] != "Alloc" || b.events[0].IPAddress != "127.0.0.1" {
		t.Errorf("observers got unexpected event: %+v %+v", a.events[0], b.events[0])
	}
}

func TestSubject_RecordNilSafe(t *testing.T) {
	var s *Subject
	req := httptest.NewRequest(http.MethodPost, "/update", nil)
	s.Record(req, []string{"Alloc"})
}

func TestFileObserver_AppendsJSONLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	obs := NewFileObserver(path)

	obs.Update(Event{TS: 42, Metrics: []string{"Alloc", "Frees"}, IPAddress: "192.168.0.42"})
	obs.Update(Event{TS: 43, Metrics: []string{"HeapSys"}, IPAddress: "10.0.0.1"})

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

func TestHTTPObserver_PostsJSON(t *testing.T) {
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

	NewHTTPObserver(srv.URL).Update(Event{
		TS:        99,
		Metrics:   []string{"PollCount"},
		IPAddress: "127.0.0.1",
	})

	if got.TS != 99 || got.IPAddress != "127.0.0.1" || len(got.Metrics) != 1 || got.Metrics[0] != "PollCount" {
		t.Errorf("posted event: %+v", got)
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
	obs := &recordingObserver{}
	s := New("", "")
	s.Attach(obs)

	req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
	req.RemoteAddr = "192.168.0.42:9"
	before := time.Now().Unix()
	s.Record(req, []string{"Alloc", "Frees"})
	after := time.Now().Unix()

	if len(obs.events) != 1 {
		t.Fatalf("want 1 event, got %d", len(obs.events))
	}
	e := obs.events[0]
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
