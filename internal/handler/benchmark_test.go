package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/middleware"
	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func populatedStore() repository.Storage {
	store := repository.NewMemStorage()
	ctx := context.Background()
	for i := 0; i < 32; i++ {
		store.UpdateGauge(ctx, fmt.Sprintf("gauge_%d", i), float64(i)*1.5)
	}
	for i := 0; i < 8; i++ {
		store.UpdateCounter(ctx, fmt.Sprintf("counter_%d", i), int64(i+1))
	}
	return store
}

func BenchmarkIndex(b *testing.B) {
	h := handler.Index(populatedStore())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}

func BenchmarkUpdateJSON(b *testing.B) {
	h := handler.UpdateJSON(repository.NewMemStorage(), nil)
	body := []byte(`{"id":"Alloc","type":"gauge","value":123.45}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}

func BenchmarkUpdatesBatch(b *testing.B) {
	batch := make([]models.Metrics, 0, 16)
	for i := 0; i < 15; i++ {
		v := float64(i)
		batch = append(batch, models.Metrics{ID: fmt.Sprintf("g%d", i), MType: models.Gauge, Value: &v})
	}
	delta := int64(1)
	batch = append(batch, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &delta})
	raw, err := json.Marshal(batch)
	if err != nil {
		b.Fatal(err)
	}

	h := handler.UpdatesBatch(repository.NewMemStorage(), nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}

func BenchmarkValueJSON(b *testing.B) {
	store := repository.NewMemStorage()
	v := 99.0
	store.UpdateGauge(context.Background(), "Alloc", v)
	h := handler.ValueJSON(store)
	body := []byte(`{"id":"Alloc","type":"gauge"}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}

func BenchmarkGzipIndex(b *testing.B) {
	h := middleware.GzipCompress(handler.Index(populatedStore()))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}
