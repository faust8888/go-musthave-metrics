package repository_test

import (
	"context"
	"testing"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func TestMemStorage_GaugeAndCounter(t *testing.T) {
	ctx := context.Background()
	s := repository.NewMemStorage()

	s.UpdateGauge(ctx, "temp", 1.1)
	s.UpdateGauge(ctx, "temp", 2.2)
	v, ok := s.GetGauge(ctx, "temp")
	if !ok || v != 2.2 {
		t.Errorf("gauge: got %v %v, want 2.2 true", v, ok)
	}
	if _, ok := s.GetGauge(ctx, "missing"); ok {
		t.Error("missing gauge should not be found")
	}

	s.UpdateCounter(ctx, "hits", 10)
	s.UpdateCounter(ctx, "hits", 5)
	d, ok := s.GetCounter(ctx, "hits")
	if !ok || d != 15 {
		t.Errorf("counter: got %v %v, want 15 true", d, ok)
	}
	if _, ok := s.GetCounter(ctx, "missing"); ok {
		t.Error("missing counter should not be found")
	}
}

func TestMemStorage_GetAll(t *testing.T) {
	ctx := context.Background()
	s := repository.NewMemStorage()
	s.UpdateGauge(ctx, "a", 1)
	s.UpdateCounter(ctx, "b", 2)

	gauges := s.GetAllGauges(ctx)
	if gauges["a"] != 1 {
		t.Errorf("gauges: %v", gauges)
	}
	gauges["a"] = 99
	if v, _ := s.GetGauge(ctx, "a"); v != 1 {
		t.Error("GetAllGauges must return a copy")
	}

	counters := s.GetAllCounters(ctx)
	if counters["b"] != 2 {
		t.Errorf("counters: %v", counters)
	}
	counters["b"] = 99
	if v, _ := s.GetCounter(ctx, "b"); v != 2 {
		t.Error("GetAllCounters must return a copy")
	}
}

func TestMemStorage_UpdateBatch(t *testing.T) {
	ctx := context.Background()
	s := repository.NewMemStorage()
	g, c := 3.5, int64(4)
	err := s.UpdateBatch(ctx, []models.Metrics{
		{ID: "g", MType: models.Gauge, Value: &g},
		{ID: "c", MType: models.Counter, Delta: &c},
		{ID: "c", MType: models.Counter, Delta: &c},
		{ID: "skip-g", MType: models.Gauge},
		{ID: "skip-c", MType: models.Counter},
	})
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := s.GetGauge(ctx, "g"); !ok || v != 3.5 {
		t.Errorf("batch gauge: %v %v", v, ok)
	}
	if d, ok := s.GetCounter(ctx, "c"); !ok || d != 8 {
		t.Errorf("batch counter: %v %v", d, ok)
	}
}
