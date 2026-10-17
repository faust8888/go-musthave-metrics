package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func TestIndexHandler(t *testing.T) {
	store := repository.NewMemStorage()
	store.UpdateGauge(context.Background(), "Alloc", 1024.5)
	store.UpdateCounter(context.Background(), "PollCount", 7)

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

func TestIndexHandler_EscapesMetricNames(t *testing.T) {
	store := repository.NewMemStorage()
	store.UpdateGauge(context.Background(), `<script>alert("g")</script>`, 1)
	store.UpdateCounter(context.Background(), `<script>alert("c")</script>`, 2)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.Index(store)(w, req)

	body := w.Body.String()
	if strings.Contains(body, `<script>`) {
		t.Fatalf("unescaped script tag in HTML: %s", body)
	}
	if !strings.Contains(body, `&lt;script&gt;alert(&#34;g&#34;)&lt;/script&gt;`) {
		t.Errorf("escaped gauge name missing: %s", body)
	}
	if !strings.Contains(body, `&lt;script&gt;alert(&#34;c&#34;)&lt;/script&gt;`) {
		t.Errorf("escaped counter name missing: %s", body)
	}
}
