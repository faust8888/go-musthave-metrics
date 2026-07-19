package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func newValueRouter(store repository.Storage) http.Handler {
	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", handler.Value(store))
	return r
}

func TestValueHandler(t *testing.T) {
	store := repository.NewMemStorage()
	store.UpdateGauge("temp", 36.6)
	store.UpdateCounter("hits", 42)

	router := newValueRouter(store)

	tests := []struct {
		name     string
		path     string
		wantCode int
		wantBody string
	}{
		{"existing gauge", "/value/gauge/temp", http.StatusOK, "36.6"},
		{"existing counter", "/value/counter/hits", http.StatusOK, "42"},
		{"unknown gauge", "/value/gauge/unknown", http.StatusNotFound, ""},
		{"unknown counter", "/value/counter/unknown", http.StatusNotFound, ""},
		{"invalid type", "/value/bad/temp", http.StatusBadRequest, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("path %s: got status %d, want %d", tt.path, w.Code, tt.wantCode)
			}
			if tt.wantBody != "" && !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Errorf("path %s: body %q does not contain %q", tt.path, w.Body.String(), tt.wantBody)
			}
		})
	}
}
