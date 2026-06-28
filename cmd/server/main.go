package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func main() {
	addr := flag.String("a", "localhost:8080", "HTTP server address")
	flag.Parse()

	store := repository.NewMemStorage()

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", handler.Index(store))
	r.Post("/update/{type}/{name}/{value}", handler.Update(store))
	r.Get("/value/{type}/{name}", handler.Value(store))

	log.Printf("Server started on %s", *addr)
	if err := http.ListenAndServe(*addr, r); err != nil {
		log.Fatal(err)
	}
}
