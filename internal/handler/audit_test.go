package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/faust8888/go-musthave-metrics/internal/audit"
	"github.com/faust8888/go-musthave-metrics/internal/handler"
	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

type captureSink struct {
	events []audit.Event
}

func (s *captureSink) Write(e audit.Event) {
	s.events = append(s.events, e)
}

func newHandlerAuditor(t *testing.T) *audit.Auditor {
	t.Helper()
	a, err := audit.New("", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return a
}

func TestUpdate_AuditsSuccessfulRequest(t *testing.T) {
	sink := &captureSink{}
	auditor := newHandlerAuditor(t)
	auditor.Add(sink)

	store := repository.NewMemStorage()
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", handler.Update(store, auditor))

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/1", nil)
	req.RemoteAddr = "192.168.0.42:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	auditor.Flush()

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	if len(sink.events) != 1 {
		t.Fatalf("want 1 audit event, got %d", len(sink.events))
	}
	if sink.events[0].Metrics[0] != "Alloc" {
		t.Errorf("metric: got %v", sink.events[0].Metrics)
	}
	if sink.events[0].IPAddress != "192.168.0.42" {
		t.Errorf("ip: got %q", sink.events[0].IPAddress)
	}
}

func TestUpdate_DoesNotAuditInvalidRequest(t *testing.T) {
	sink := &captureSink{}
	auditor := newHandlerAuditor(t)
	auditor.Add(sink)

	store := repository.NewMemStorage()
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", handler.Update(store, auditor))
	req := httptest.NewRequest(http.MethodPost, "/update/unknown/x/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	auditor.Flush()

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", w.Code)
	}
	if len(sink.events) != 0 {
		t.Errorf("invalid request must not be audited, got %d events", len(sink.events))
	}
}

func TestUpdatesBatch_AuditsMetricNames(t *testing.T) {
	sink := &captureSink{}
	auditor := newHandlerAuditor(t)
	auditor.Add(sink)

	body, _ := json.Marshal([]models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: ptr(1.0)},
		{ID: "Frees", MType: models.Gauge, Value: ptr(2.0)},
	})
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.UpdatesBatch(repository.NewMemStorage(), auditor)(w, req)
	auditor.Flush()

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	if len(sink.events) != 1 {
		t.Fatalf("want 1 event, got %d", len(sink.events))
	}
	got := sink.events[0].Metrics
	if len(got) != 2 || got[0] != "Alloc" || got[1] != "Frees" {
		t.Errorf("metrics: got %v", got)
	}
}
