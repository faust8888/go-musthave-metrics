package repository

import (
	"context"
	"sync"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
)

type Storage interface {
	UpdateGauge(ctx context.Context, name string, value float64)
	UpdateCounter(ctx context.Context, name string, value int64)
	GetGauge(ctx context.Context, name string) (float64, bool)
	GetCounter(ctx context.Context, name string) (int64, bool)
	GetAllGauges(ctx context.Context) map[string]float64
	GetAllCounters(ctx context.Context) map[string]int64
	// UpdateBatch applies all metrics atomically (or as close as possible).
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
}

type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *MemStorage) UpdateGauge(_ context.Context, name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

func (s *MemStorage) UpdateCounter(_ context.Context, name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
}

func (s *MemStorage) GetGauge(_ context.Context, name string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.gauges[name]
	return v, ok
}

func (s *MemStorage) GetCounter(_ context.Context, name string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.counters[name]
	return v, ok
}

func (s *MemStorage) GetAllGauges(_ context.Context) map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		out[k] = v
	}
	return out
}

func (s *MemStorage) GetAllCounters(_ context.Context) map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]int64, len(s.counters))
	for k, v := range s.counters {
		out[k] = v
	}
	return out
}

func (s *MemStorage) UpdateBatch(_ context.Context, metrics []models.Metrics) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value != nil {
				s.gauges[m.ID] = *m.Value
			}
		case models.Counter:
			if m.Delta != nil {
				s.counters[m.ID] += *m.Delta
			}
		}
	}
	return nil
}
