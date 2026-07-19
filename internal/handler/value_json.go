package handler

import (
	"encoding/json"
	"net/http"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func ValueJSON(store repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m models.Metrics
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		switch m.MType {
		case models.Gauge:
			v, ok := store.GetGauge(m.ID)
			if !ok {
				http.Error(w, "metric not found", http.StatusNotFound)
				return
			}
			m.Value = &v
		case models.Counter:
			v, ok := store.GetCounter(m.ID)
			if !ok {
				http.Error(w, "metric not found", http.StatusNotFound)
				return
			}
			m.Delta = &v
		default:
			http.Error(w, "invalid metric type", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m)
	}
}
