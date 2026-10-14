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

type captureObserver struct {
	events []audit.Event
}

func (o *captureObserver) Update(e audit.Event) {
	o.events = append(o.events, e)
}

func TestUpdate_AuditsSuccessfulRequest(t *testing.T) {
	obs := &captureObserver{}
	auditor := audit.New("", "")
	auditor.Attach(obs)

	store := repository.NewMemStorage()
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", handler.Update(store, auditor))

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/1", nil)
	req.RemoteAddr = "192.168.0.42:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	if len(obs.events) != 1 {
		t.Fatalf("want 1 audit event, got %d", len(obs.events))
	}
	if obs.events[0].Metrics[0] != "Alloc" {
		t.Errorf("metric: got %v", obs.events[0].Metrics)
	}
	if obs.events[0].IPAddress != "192.168.0.42" {
		t.Errorf("ip: got %q", obs.events[0].IPAddress)
	}
}

func TestUpdate_DoesNotAuditInvalidRequest(t *testing.T) {
	obs := &captureObserver{}
	auditor := audit.New("", "")
	auditor.Attach(obs)

	store := repository.NewMemStorage()
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", handler.Update(store, auditor))
	req := httptest.NewRequest(http.MethodPost, "/update/unknown/x/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", w.Code)
	}
	if len(obs.events) != 0 {
		t.Errorf("invalid request must not be audited, got %d events", len(obs.events))
	}
}

func TestUpdatesBatch_AuditsMetricNames(t *testing.T) {
	obs := &captureObserver{}
	auditor := audit.New("", "")
	auditor.Attach(obs)

	body, _ := json.Marshal([]models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: ptr(1.0)},
		{ID: "Frees", MType: models.Gauge, Value: ptr(2.0)},
	})
	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.UpdatesBatch(repository.NewMemStorage(), auditor)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	if len(obs.events) != 1 {
		t.Fatalf("want 1 event, got %d", len(obs.events))
	}
	got := obs.events[0].Metrics
	if len(got) != 2 || got[0] != "Alloc" || got[1] != "Frees" {
		t.Errorf("metrics: got %v", got)
	}
}
