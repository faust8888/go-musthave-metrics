package main

import (
	"log"
	"net/http"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func main() {
	store := repository.NewMemStorage()

	mux := http.NewServeMux()
	mux.HandleFunc("/update/", handler.Update(store))

	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
