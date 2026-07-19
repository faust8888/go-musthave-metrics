package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/middleware"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func main() {
	addr := flag.String("a", "localhost:8080", "HTTP server address")
	storeInterval := flag.Int("i", 300, "store interval in seconds (0 = sync)")
	filePath := flag.String("f", "/tmp/metrics-db.json", "file storage path")
	restore := flag.Bool("r", true, "restore metrics from file on start")
	flag.Parse()

	if v := os.Getenv("ADDRESS"); v != "" {
		*addr = v
	}
	if v := os.Getenv("STORE_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*storeInterval = n
		}
	}
	if v := os.Getenv("FILE_STORAGE_PATH"); v != "" {
		*filePath = v
	}
	if v := os.Getenv("RESTORE"); v != "" {
		*restore = v == "true"
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	var store repository.Storage
	if *filePath != "" {
		fs := repository.NewFileStorage(*filePath, *storeInterval == 0)
		if *restore {
			if err := fs.Load(); err != nil {
				logger.Error("failed to load metrics from file", zap.Error(err))
			} else {
				logger.Info("metrics loaded from file", zap.String("path", *filePath))
			}
		}
		store = fs

		if *storeInterval > 0 {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			go func() {
				ticker := time.NewTicker(time.Duration(*storeInterval) * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						if err := fs.Save(); err != nil {
							logger.Error("failed to save metrics", zap.Error(err))
						} else {
							logger.Info("metrics saved", zap.String("path", *filePath))
						}
					case <-ctx.Done():
						return
					}
				}
			}()
		}
	} else {
		store = repository.NewMemStorage()
	}

	r := chi.NewRouter()
	r.Use(middleware.GzipDecompress)
	r.Use(middleware.GzipCompress)
	r.Use(middleware.Logger(logger))

	r.Get("/", handler.Index(store))
	r.Post("/update/{type}/{name}/{value}", handler.Update(store))
	r.Get("/value/{type}/{name}", handler.Value(store))
	r.Post("/update", handler.UpdateJSON(store))
	r.Post("/value", handler.ValueJSON(store))

	logger.Info("Server started", zap.String("address", *addr))
	if err := http.ListenAndServe(*addr, r); err != nil {
		log.Fatal(err)
	}
}
