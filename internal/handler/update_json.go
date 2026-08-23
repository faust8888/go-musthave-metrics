package handler

import (
	"encoding/json"
	"net/http"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func UpdateJSON(store repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m models.Metrics
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		switch m.MType {
		case models.Gauge:
			if m.Value == nil {
				http.Error(w, "value is required for gauge", http.StatusBadRequest)
				return
			}
			store.UpdateGauge(ctx, m.ID, *m.Value)
		case models.Counter:
			if m.Delta == nil {
				http.Error(w, "delta is required for counter", http.StatusBadRequest)
				return
			}
			store.UpdateCounter(ctx, m.ID, *m.Delta)
			v, _ := store.GetCounter(ctx, m.ID)
			m.Delta = &v
		default:
			http.Error(w, "invalid metric type", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m)
	}
}
