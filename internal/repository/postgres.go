package repository

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"

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

func (s *PostgresStorage) UpdateGauge(ctx context.Context, name string, value float64) {
	_ = retry.Do(func() error {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO gauges(name, value) VALUES($1, $2)
			 ON CONFLICT(name) DO UPDATE SET value = EXCLUDED.value`,
			name, value)
		return err
	}, isPgConnError, retry.DefaultDelays)
}

func (s *PostgresStorage) UpdateCounter(ctx context.Context, name string, value int64) {
	_ = retry.Do(func() error {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO counters(name, delta) VALUES($1, $2)
			 ON CONFLICT(name) DO UPDATE SET delta = counters.delta + EXCLUDED.delta`,
			name, value)
		return err
	}, isPgConnError, retry.DefaultDelays)
}

func (s *PostgresStorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	var v float64
	err := retry.Do(func() error {
		return s.db.QueryRowContext(ctx,
			`SELECT value FROM gauges WHERE name = $1`, name).Scan(&v)
	}, isPgConnError, retry.DefaultDelays)
	if err != nil {
		return 0, false
	}
	return v, true
}

func (s *PostgresStorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	var d int64
	err := retry.Do(func() error {
		return s.db.QueryRowContext(ctx,
			`SELECT delta FROM counters WHERE name = $1`, name).Scan(&d)
	}, isPgConnError, retry.DefaultDelays)
	if err != nil {
		return 0, false
	}
	return d, true
}

func (s *PostgresStorage) GetAllGauges(ctx context.Context) map[string]float64 {
	var out map[string]float64
	_ = retry.Do(func() error {
		rows, err := s.db.QueryContext(ctx, `SELECT name, value FROM gauges`)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = make(map[string]float64)
		for rows.Next() {
			var name string
			var value float64
			if err := rows.Scan(&name, &value); err != nil {
				return err
			}
			out[name] = value
		}
		return rows.Err()
	}, isPgConnError, retry.DefaultDelays)
	if out == nil {
		return map[string]float64{}
	}
	return out
}

func (s *PostgresStorage) GetAllCounters(ctx context.Context) map[string]int64 {
	var out map[string]int64
	_ = retry.Do(func() error {
		rows, err := s.db.QueryContext(ctx, `SELECT name, delta FROM counters`)
		if err != nil {
			return err
		}
		defer rows.Close()
		out = make(map[string]int64)
		for rows.Next() {
			var name string
			var delta int64
			if err := rows.Scan(&name, &delta); err != nil {
				return err
			}
			out[name] = delta
		}
		return rows.Err()
	}, isPgConnError, retry.DefaultDelays)
	if out == nil {
		return map[string]int64{}
	}
	return out
}

func (s *PostgresStorage) UpdateBatch(ctx context.Context, metrics []models.Metrics) error {
	return retry.Do(func() error {
		return s.execBatch(ctx, metrics)
	}, isPgConnError, retry.DefaultDelays)
}

func (s *PostgresStorage) execBatch(ctx context.Context, metrics []models.Metrics) error {
	// Deduplicate within the batch: PostgreSQL's ON CONFLICT DO UPDATE cannot
	// handle the same key appearing twice in a single VALUES list.
	// For gauges: last value wins. For counters: accumulate deltas.
	gaugeMap := make(map[string]float64)
	counterMap := make(map[string]int64)
	// Preserve insertion order for deterministic queries.
	var gaugeOrder []string
	var counterOrder []string

	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				continue
			}
			if _, seen := gaugeMap[m.ID]; !seen {
				gaugeOrder = append(gaugeOrder, m.ID)
			}
			gaugeMap[m.ID] = *m.Value
		case models.Counter:
			if m.Delta == nil {
				continue
			}
			if _, seen := counterMap[m.ID]; !seen {
				counterOrder = append(counterOrder, m.ID)
			}
			counterMap[m.ID] += *m.Delta
		default:
			return fmt.Errorf("unknown metric type %q", m.MType)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if len(gaugeOrder) > 0 {
		placeholders := make([]string, len(gaugeOrder))
		args := make([]any, 0, len(gaugeOrder)*2)
		for i, name := range gaugeOrder {
			placeholders[i] = fmt.Sprintf("($%d,$%d)", i*2+1, i*2+2)
			args = append(args, name, gaugeMap[name])
		}
		q := "INSERT INTO gauges(name,value) VALUES " + strings.Join(placeholders, ",") +
			" ON CONFLICT(name) DO UPDATE SET value = EXCLUDED.value"
		if _, err = tx.ExecContext(ctx, q, args...); err != nil {
			return err
		}
	}

	if len(counterOrder) > 0 {
		placeholders := make([]string, len(counterOrder))
		args := make([]any, 0, len(counterOrder)*2)
		for i, name := range counterOrder {
			placeholders[i] = fmt.Sprintf("($%d,$%d)", i*2+1, i*2+2)
			args = append(args, name, counterMap[name])
		}
		q := "INSERT INTO counters(name,delta) VALUES " + strings.Join(placeholders, ",") +
			" ON CONFLICT(name) DO UPDATE SET delta = counters.delta + EXCLUDED.delta"
		if _, err = tx.ExecContext(ctx, q, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}
