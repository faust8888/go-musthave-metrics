package handler

import (
	"encoding/json"
	"net/http"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func UpdatesJSON(store repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var metrics []models.Metrics
		if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if len(metrics) == 0 {
			http.Error(w, "empty batch", http.StatusBadRequest)
			return
		}
		for _, m := range metrics {
			switch m.MType {
			case models.Gauge:
				if m.Value == nil {
					http.Error(w, "value is required for gauge", http.StatusBadRequest)
					return
				}
			case models.Counter:
				if m.Delta == nil {
					http.Error(w, "delta is required for counter", http.StatusBadRequest)
					return
				}
			default:
				http.Error(w, "invalid metric type: "+m.MType, http.StatusBadRequest)
				return
			}
		}
		if err := store.UpdateBatch(metrics); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
