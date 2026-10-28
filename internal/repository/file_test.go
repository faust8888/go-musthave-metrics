package repository_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func TestFileStorage_SaveLoad(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "metrics.json")
	fs := repository.NewFileStorage(path, false)

	fs.UpdateGauge(ctx, "temp", 36.6)
	fs.UpdateCounter(ctx, "hits", 7)
	if err := fs.Save(); err != nil {
		t.Fatal(err)
	}

	loaded := repository.NewFileStorage(path, false)
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	if v, ok := loaded.GetGauge(ctx, "temp"); !ok || v != 36.6 {
		t.Errorf("loaded gauge: %v %v", v, ok)
	}
	if d, ok := loaded.GetCounter(ctx, "hits"); !ok || d != 7 {
		t.Errorf("loaded counter: %v %v", d, ok)
	}
}

func TestFileStorage_LoadMissingFile(t *testing.T) {
	fs := repository.NewFileStorage(filepath.Join(t.TempDir(), "no-such.json"), false)
	if err := fs.Load(); err != nil {
		t.Fatalf("missing file must be ok, got %v", err)
	}
}

func TestFileStorage_SyncWrite(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "sync.json")
	fs := repository.NewFileStorage(path, true)

	fs.UpdateGauge(ctx, "g", 1)
	fs.UpdateCounter(ctx, "c", 2)
	g, c := 3.0, int64(4)
	if err := fs.UpdateBatch(ctx, []models.Metrics{
		{ID: "g2", MType: models.Gauge, Value: &g},
		{ID: "c2", MType: models.Counter, Delta: &c},
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("sync write did not create file: %v", err)
	}

	loaded := repository.NewFileStorage(path, false)
	if err := loaded.Load(); err != nil {
		t.Fatal(err)
	}
	if v, _ := loaded.GetGauge(ctx, "g"); v != 1 {
		t.Errorf("g: %v", v)
	}
	if d, _ := loaded.GetCounter(ctx, "c"); d != 2 {
		t.Errorf("c: %v", d)
	}
}

func TestFileStorage_LoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	fs := repository.NewFileStorage(path, false)
	if err := fs.Load(); err == nil {
		t.Fatal("expected error for invalid json")
	}
}
