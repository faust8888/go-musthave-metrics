package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func main() {
	store := repository.NewMemStorage()

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", handler.Index(store))
	r.Post("/update/{type}/{name}/{value}", handler.Update(store))
	r.Get("/value/{type}/{name}", handler.Value(store))

	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
