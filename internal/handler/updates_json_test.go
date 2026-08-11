package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func ptr[T any](v T) *T { return &v }

func TestUpdatesJSONHandler(t *testing.T) {
	tests := []struct {
		name     string
		body     any
		wantCode int
	}{
		{
			name: "valid gauge and counter",
			body: []models.Metrics{
				{ID: "cpu", MType: models.Gauge, Value: ptr(42.5)},
				{ID: "hits", MType: models.Counter, Delta: ptr(int64(10))},
			},
			wantCode: http.StatusOK,
		},
		{
			name:     "empty batch",
			body:     []models.Metrics{},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid JSON",
			body:     "not json",
			wantCode: http.StatusBadRequest,
		},
		{
			name: "gauge without value",
			body: []models.Metrics{
				{ID: "cpu", MType: models.Gauge},
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "counter without delta",
			body: []models.Metrics{
				{ID: "hits", MType: models.Counter},
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "unknown metric type",
			body: []models.Metrics{
				{ID: "x", MType: "unknown"},
			},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := repository.NewMemStorage()

			var body []byte
			var err error
			if s, ok := tt.body.(string); ok {
				body = []byte(s)
			} else {
				body, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("marshal body: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.UpdatesBatch(store)(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("got status %d, want %d", w.Code, tt.wantCode)
			}
		})
	}
}

func TestUpdatesJSONHandler_StorageEffect(t *testing.T) {
	store := repository.NewMemStorage()

	batch := []models.Metrics{
		{ID: "temp", MType: models.Gauge, Value: ptr(36.6)},
		{ID: "hits", MType: models.Counter, Delta: ptr(int64(10))},
		{ID: "hits", MType: models.Counter, Delta: ptr(int64(20))},
	}
	body, _ := json.Marshal(batch)

	req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdatesBatch(store)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", w.Code)
	}

	if v, ok := store.GetGauge(req.Context(), "temp"); !ok || v != 36.6 {
		t.Errorf("gauge temp: got %f %v, want 36.6 true", v, ok)
	}
	if d, ok := store.GetCounter(req.Context(), "hits"); !ok || d != 30 {
		t.Errorf("counter hits: got %d %v, want 30 true", d, ok)
	}
}
