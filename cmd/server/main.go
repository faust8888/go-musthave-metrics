package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"github.com/faust8888/go-musthave-metrics/internal/envconfig"
	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/middleware"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func main() {
	addr := flag.String("a", "localhost:8080", "HTTP server address")
	storeInterval := flag.Int("i", 300, "store interval in seconds (0 = sync)")
	filePath := flag.String("f", "/tmp/metrics-db.json", "file storage path")
	restore := flag.Bool("r", true, "restore metrics from file on start")
	dsn := flag.String("d", "", "PostgreSQL DSN (DATABASE_DSN)")
	flag.Parse()

	envconfig.String("ADDRESS", addr)
	envconfig.String("FILE_STORAGE_PATH", filePath)
	envconfig.String("DATABASE_DSN", dsn)
	if err := envconfig.Int("STORE_INTERVAL", storeInterval); err != nil {
		log.Fatal(err)
	}
	if err := envconfig.Bool("RESTORE", restore); err != nil {
		log.Fatal(err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var db *sql.DB
	if *dsn != "" {
		db, err = sql.Open("pgx", *dsn)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		if err = repository.RunMigrations(db); err != nil {
			log.Fatal("migrations failed: ", err)
		}
		logger.Info("database ready", zap.String("dsn", *dsn))
	}

	store, cleanup := newStorage(ctx, db, *filePath, *storeInterval, *restore, logger)
	defer cleanup()

	r := chi.NewRouter()
	r.Use(middleware.GzipDecompress)
	r.Use(middleware.GzipCompress)
	r.Use(middleware.Logger(logger))

	r.Get("/ping", handler.Ping(db))
	r.Get("/", handler.Index(store))
	r.Post("/update/{type}/{name}/{value}", handler.Update(store))
	r.Get("/value/{type}/{name}", handler.Value(store))
	r.Post("/update", handler.UpdateJSON(store))
	r.Post("/update/", handler.UpdateJSON(store))
	r.Post("/updates/", handler.UpdatesBatch(store))
	r.Post("/value", handler.ValueJSON(store))
	r.Post("/value/", handler.ValueJSON(store))

	srv := &http.Server{Addr: *addr, Handler: r}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			logger.Error("shutdown error", zap.Error(err))
		}
	}()

	logger.Info("Server started", zap.String("address", *addr))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
