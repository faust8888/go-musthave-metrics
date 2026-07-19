package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func Update(store repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")
		metricValue := chi.URLParam(r, "value")

		switch metricType {
		case models.Gauge:
			v, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				http.Error(w, "invalid gauge value", http.StatusBadRequest)
				return
			}
			store.UpdateGauge(metricName, v)
		case models.Counter:
			v, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				http.Error(w, "invalid counter value", http.StatusBadRequest)
				return
			}
			store.UpdateCounter(metricName, v)
		default:
			http.Error(w, "invalid metric type", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
