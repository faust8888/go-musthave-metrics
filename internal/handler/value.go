package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func Value(store repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")

		switch metricType {
		case models.Gauge:
			v, ok := store.GetGauge(metricName)
			if !ok {
				http.Error(w, "metric not found", http.StatusNotFound)
				return
			}
			fmt.Fprint(w, strconv.FormatFloat(v, 'f', -1, 64))
		case models.Counter:
			v, ok := store.GetCounter(metricName)
			if !ok {
				http.Error(w, "metric not found", http.StatusNotFound)
				return
			}
			fmt.Fprintf(w, "%d", v)
		default:
			http.Error(w, "invalid metric type", http.StatusBadRequest)
		}
	}
}
