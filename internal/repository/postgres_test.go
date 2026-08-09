package repository_test

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN not set, skipping postgres integration tests")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := repository.RunMigrations(db); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestPostgresStorage_Gauge(t *testing.T) {
	db := openTestDB(t)
	s := repository.NewPostgresStorage(db)

	s.UpdateGauge("cpu", 42.5)
	v, ok := s.GetGauge("cpu")
	if !ok {
		t.Fatal("gauge not found after update")
	}
	if v != 42.5 {
		t.Errorf("got %f, want 42.5", v)
	}

	// overwrite
	s.UpdateGauge("cpu", 99.9)
	v, ok = s.GetGauge("cpu")
	if !ok || v != 99.9 {
		t.Errorf("got %f %v, want 99.9 true", v, ok)
	}

	// missing key
	_, ok = s.GetGauge("nonexistent")
	if ok {
		t.Error("expected false for nonexistent gauge")
	}
}

func TestPostgresStorage_Counter(t *testing.T) {
	db := openTestDB(t)
	s := repository.NewPostgresStorage(db)

	s.UpdateCounter("hits", 10)
	s.UpdateCounter("hits", 20)
	d, ok := s.GetCounter("hits")
	if !ok {
		t.Fatal("counter not found after update")
	}
	if d != 30 {
		t.Errorf("got %d, want 30", d)
	}

	// missing key
	_, ok = s.GetCounter("nonexistent")
	if ok {
		t.Error("expected false for nonexistent counter")
	}
}

func TestPostgresStorage_GetAll(t *testing.T) {
	db := openTestDB(t)
	s := repository.NewPostgresStorage(db)

	s.UpdateGauge("temp", 36.6)
	s.UpdateCounter("reqs", 5)

	gauges := s.GetAllGauges()
	if _, ok := gauges["temp"]; !ok {
		t.Error("temp gauge missing from GetAllGauges")
	}

	counters := s.GetAllCounters()
	if _, ok := counters["reqs"]; !ok {
		t.Error("reqs counter missing from GetAllCounters")
	}
}
