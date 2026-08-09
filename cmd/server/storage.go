package main

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"

	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

// newStorage selects the appropriate Storage backend:
//   - PostgresStorage when db is non-nil (migrations are applied by the caller)
//   - FileStorage when filePath is non-empty
//   - MemStorage otherwise
//
// The returned cleanup function must be called on shutdown.
func newStorage(ctx context.Context, db *sql.DB, filePath string, storeInterval int, restore bool, logger *zap.Logger) (repository.Storage, func()) {
	if db != nil {
		return repository.NewPostgresStorage(db), func() {}
	}

	if filePath == "" {
		return repository.NewMemStorage(), func() {}
	}

	fs := repository.NewFileStorage(filePath, storeInterval == 0)
	if restore {
		if err := fs.Load(); err != nil {
			logger.Error("failed to load metrics from file", zap.Error(err))
		} else {
			logger.Info("metrics loaded from file", zap.String("path", filePath))
		}
	}

	if storeInterval <= 0 {
		return fs, func() {}
	}

	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Duration(storeInterval) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := fs.Save(); err != nil {
					logger.Error("failed to save metrics", zap.Error(err))
				} else {
					logger.Info("metrics saved", zap.String("path", filePath))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return fs, func() {
		cancel()
		<-done
	}
}
