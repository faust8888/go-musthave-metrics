package repository

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/retry"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations applies all pending up-migrations embedded in the binary.
func RunMigrations(db *sql.DB) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		return err
	}
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// isPgConnError returns true for PostgreSQL Class 08 (Connection Exception) errors.
func isPgConnError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgerrcode.IsConnectionException(pgErr.Code)
}

// PostgresStorage implements Storage using a PostgreSQL database.
type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

func (s *PostgresStorage) UpdateGauge(name string, value float64) {
	_ = retry.Do(func() error {
		_, err := s.db.ExecContext(context.Background(),
			`INSERT INTO gauges(name, value) VALUES($1, $2)
			 ON CONFLICT(name) DO UPDATE SET value = EXCLUDED.value`,
			name, value)
		return err
	}, isPgConnError)
}

func (s *PostgresStorage) UpdateCounter(name string, value int64) {
	_ = retry.Do(func() error {
		_, err := s.db.ExecContext(context.Background(),
			`INSERT INTO counters(name, delta) VALUES($1, $2)
			 ON CONFLICT(name) DO UPDATE SET delta = counters.delta + EXCLUDED.delta`,
			name, value)
		return err
	}, isPgConnError)
}

func (s *PostgresStorage) GetGauge(name string) (float64, bool) {
	var v float64
	err := retry.Do(func() error {
		return s.db.QueryRowContext(context.Background(),
			`SELECT value FROM gauges WHERE name = $1`, name).Scan(&v)
	}, isPgConnError)
	if err != nil {
		return 0, false
	}
	return v, true
}

func (s *PostgresStorage) GetCounter(name string) (int64, bool) {
	var d int64
	err := retry.Do(func() error {
		return s.db.QueryRowContext(context.Background(),
			`SELECT delta FROM counters WHERE name = $1`, name).Scan(&d)
	}, isPgConnError)
	if err != nil {
		return 0, false
	}
	return d, true
}

func (s *PostgresStorage) GetAllGauges() map[string]float64 {
	var out map[string]float64
	_ = retry.Do(func() error {
		rows, err := s.db.QueryContext(context.Background(), `SELECT name, value FROM gauges`)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = make(map[string]float64)
		for rows.Next() {
			var name string
			var value float64
			if err := rows.Scan(&name, &value); err == nil {
				out[name] = value
			}
		}
		return rows.Err()
	}, isPgConnError)
	if out == nil {
		return map[string]float64{}
	}
	return out
}

func (s *PostgresStorage) GetAllCounters() map[string]int64 {
	var out map[string]int64
	_ = retry.Do(func() error {
		rows, err := s.db.QueryContext(context.Background(), `SELECT name, delta FROM counters`)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = make(map[string]int64)
		for rows.Next() {
			var name string
			var delta int64
			if err := rows.Scan(&name, &delta); err == nil {
				out[name] = delta
			}
		}
		return rows.Err()
	}, isPgConnError)
	if out == nil {
		return map[string]int64{}
	}
	return out
}

func (s *PostgresStorage) UpdateBatch(metrics []models.Metrics) error {
	return retry.Do(func() error {
		return s.execBatch(metrics)
	}, isPgConnError)
}

func (s *PostgresStorage) execBatch(metrics []models.Metrics) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				continue
			}
			_, err = tx.ExecContext(context.Background(),
				`INSERT INTO gauges(name, value) VALUES($1, $2)
				 ON CONFLICT(name) DO UPDATE SET value = EXCLUDED.value`,
				m.ID, *m.Value)
		case models.Counter:
			if m.Delta == nil {
				continue
			}
			_, err = tx.ExecContext(context.Background(),
				`INSERT INTO counters(name, delta) VALUES($1, $2)
				 ON CONFLICT(name) DO UPDATE SET delta = counters.delta + EXCLUDED.delta`,
				m.ID, *m.Delta)
		default:
			return fmt.Errorf("unknown metric type %q", m.MType)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
