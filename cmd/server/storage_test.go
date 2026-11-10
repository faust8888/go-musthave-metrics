package main

import (
	"context"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func TestNewStorage_SaveOnCleanup(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "metrics.json")
	store, cleanup := newStorage(ctx, nil, path, 300, false, zap.NewNop())

	store.UpdateGauge(ctx, "Alloc", 42)
	store.UpdateCounter(ctx, "PollCount", 3)
	cleanup()

	loaded := repository.NewFileStorage(path, false)
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	if v, ok := loaded.GetGauge(ctx, "Alloc"); !ok || v != 42 {
		t.Errorf("gauge after shutdown: %v %v", v, ok)
	}
	if d, ok := loaded.GetCounter(ctx, "PollCount"); !ok || d != 3 {
		t.Errorf("counter after shutdown: %v %v", d, ok)
	}
}

func TestNewStorage_SaveOnCleanupSync(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "sync.json")
	store, cleanup := newStorage(ctx, nil, path, 0, false, zap.NewNop())
	store.UpdateGauge(ctx, "g", 1)
	cleanup()

	loaded := repository.NewFileStorage(path, false)
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	if v, ok := loaded.GetGauge(ctx, "g"); !ok || v != 1 {
		t.Errorf("gauge after shutdown: %v %v", v, ok)
	}
}
