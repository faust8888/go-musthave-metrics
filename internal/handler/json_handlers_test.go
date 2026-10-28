package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func TestUpdateJSONHandler(t *testing.T) {
	store := repository.NewMemStorage()
	h := handler.UpdateJSON(store, nil)

	gauge := 12.5
	body, _ := json.Marshal(models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &gauge})
	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("gauge status: %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: %q", ct)
	}

	delta := int64(3)
	body, _ = json.Marshal(models.Metrics{ID: "hits", MType: models.Counter, Delta: &delta})
	req = httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("counter status: %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader([]byte("{")))
	w = httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid json: got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader([]byte(`{"id":"x","type":"other"}`)))
	w = httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad type: got %d", w.Code)
	}
}

func TestValueJSONHandler(t *testing.T) {
	store := repository.NewMemStorage()
	store.UpdateGauge(context.Background(), "Alloc", 9)
	store.UpdateCounter(context.Background(), "hits", 4)
	h := handler.ValueJSON(store)

	body, _ := json.Marshal(models.Metrics{ID: "Alloc", MType: models.Gauge})
	req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: %d", w.Code)
	}

	body, _ = json.Marshal(models.Metrics{ID: "hits", MType: models.Counter})
	req = httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("counter status: %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader([]byte(`{"id":"nope","type":"gauge"}`)))
	w = httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("missing: got %d, want 404", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader([]byte(`{"id":"x","type":"other"}`)))
	w = httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad type: got %d", w.Code)
	}
}
