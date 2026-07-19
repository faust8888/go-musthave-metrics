package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func TestIndexHandler(t *testing.T) {
	store := repository.NewMemStorage()
	store.UpdateGauge("Alloc", 1024.5)
	store.UpdateCounter("PollCount", 7)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.Index(store)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Alloc") {
		t.Error("response does not contain gauge name 'Alloc'")
	}
	if !strings.Contains(body, "PollCount") {
		t.Error("response does not contain counter name 'PollCount'")
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type: got %q, want text/html", ct)
	}
}
