package handler

import (
	"encoding/json"
	"io"
	"net/http"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

// ValueJSON handles POST /value and returns the requested metric as JSON.
func ValueJSON(store repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		var m models.Metrics
		if err := json.Unmarshal(body, &m); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		switch m.MType {
		case models.Gauge:
			v, ok := store.GetGauge(ctx, m.ID)
			if !ok {
				http.Error(w, "metric not found", http.StatusNotFound)
				return
			}
			m.Value = &v
		case models.Counter:
			v, ok := store.GetCounter(ctx, m.ID)
			if !ok {
				http.Error(w, "metric not found", http.StatusNotFound)
				return
			}
			m.Delta = &v
		default:
			http.Error(w, "invalid metric type", http.StatusBadRequest)
			return
		}

		writeJSON(w, m)
	}
}
