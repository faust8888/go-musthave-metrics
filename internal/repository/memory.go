// Package repository stores metrics in memory, on disk or in PostgreSQL.
package repository

import (
	"context"
	"sync"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
)

// Storage is the persistence API used by HTTP handlers.
type Storage interface {
	// UpdateGauge replaces the gauge with the given name.
	UpdateGauge(ctx context.Context, name string, value float64)
	// UpdateCounter adds value to the counter with the given name.
	UpdateCounter(ctx context.Context, name string, value int64)
	// GetGauge returns the gauge value and whether it exists.
	GetGauge(ctx context.Context, name string) (float64, bool)
	// GetCounter returns the counter value and whether it exists.
	GetCounter(ctx context.Context, name string) (int64, bool)
	// GetAllGauges returns all stored gauges.
	GetAllGauges(ctx context.Context) map[string]float64
	// GetAllCounters returns all stored counters.
	GetAllCounters(ctx context.Context) map[string]int64
	// UpdateBatch applies all metrics atomically (or as close as possible).
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
}

// MemStorage keeps gauges and counters in process memory.
type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewMemStorage returns an empty in-memory store.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateGauge replaces the gauge with the given name.
func (s *MemStorage) UpdateGauge(_ context.Context, name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gauges[name] = value
}

// UpdateCounter adds value to the counter with the given name.
func (s *MemStorage) UpdateCounter(_ context.Context, name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counters[name] += value
}

// GetGauge returns the gauge value and whether it exists.
func (s *MemStorage) GetGauge(_ context.Context, name string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.gauges[name]
	return v, ok
}

// GetCounter returns the counter value and whether it exists.
func (s *MemStorage) GetCounter(_ context.Context, name string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.counters[name]
	return v, ok
}

// GetAllGauges returns a copy of all gauges.
func (s *MemStorage) GetAllGauges(_ context.Context) map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]float64, len(s.gauges))
	for k, v := range s.gauges {
		out[k] = v
	}
	return out
}

// GetAllCounters returns a copy of all counters.
func (s *MemStorage) GetAllCounters(_ context.Context) map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]int64, len(s.counters))
	for k, v := range s.counters {
		out[k] = v
	}
	return out
}

// UpdateBatch applies all metrics under a single lock.
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
