package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func newUpdateRouter(store repository.Storage) http.Handler {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", handler.Update(store))
	return r
}

func TestUpdateHandler(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		wantCode int
	}{
		{"valid counter", http.MethodPost, "/update/counter/someMetric/527", http.StatusOK},
		{"valid gauge", http.MethodPost, "/update/gauge/temp/36.6", http.StatusOK},
		{"invalid type", http.MethodPost, "/update/unknown/test/1", http.StatusBadRequest},
		{"no metric name", http.MethodPost, "/update/counter/", http.StatusNotFound},
		{"invalid counter value", http.MethodPost, "/update/counter/test/abc", http.StatusBadRequest},
		{"invalid gauge value", http.MethodPost, "/update/gauge/test/abc", http.StatusBadRequest},
		{"wrong method GET", http.MethodGet, "/update/counter/test/1", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := repository.NewMemStorage()
			router := newUpdateRouter(store)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("path %s: got status %d, want %d", tt.path, w.Code, tt.wantCode)
			}
		})
	}
}

func TestUpdateHandlerStorage(t *testing.T) {
	store := repository.NewMemStorage()
	router := newUpdateRouter(store)

	// counter accumulates
	for _, val := range []string{"10", "20"} {
		req := httptest.NewRequest(http.MethodPost, "/update/counter/hits/"+val, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	}
	if v, ok := store.GetCounter("hits"); !ok || v != 30 {
		t.Errorf("counter hits: got %d, want 30", v)
	}

	// gauge replaces
	for _, val := range []string{"1.1", "2.2"} {
		req := httptest.NewRequest(http.MethodPost, "/update/gauge/temp/"+val, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
	if v, ok := store.GetGauge("temp"); !ok || v != 2.2 {
		t.Errorf("gauge temp: got %f, want 2.2", v)
	}
}
