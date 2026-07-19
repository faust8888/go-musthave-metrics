package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/middleware"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func main() {
	addr := flag.String("a", "localhost:8080", "HTTP server address")
	flag.Parse()

	if v := os.Getenv("ADDRESS"); v != "" {
		*addr = v
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	store := repository.NewMemStorage()

	r := chi.NewRouter()
	r.Use(middleware.Logger(logger))

	r.Get("/", handler.Index(store))
	r.Post("/update/{type}/{name}/{value}", handler.Update(store))
	r.Get("/value/{type}/{name}", handler.Value(store))

	logger.Info("Server started", zap.String("address", *addr))
	if err := http.ListenAndServe(*addr, r); err != nil {
		log.Fatal(err)
	}
}
