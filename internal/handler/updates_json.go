package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/faust8888/go-musthave-metrics/internal/audit"
	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

// UpdatesBatch handles POST /updates/ and stores a JSON array of metrics.
func UpdatesBatch(store repository.Storage, auditor *audit.Subject) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		var metrics []models.Metrics
		if err := json.Unmarshal(body, &metrics); err != nil {
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
		if err := store.UpdateBatch(r.Context(), metrics); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		names := make([]string, 0, len(metrics))
		for _, m := range metrics {
			names = append(names, m.ID)
		}
		auditor.Record(r, names)
		w.WriteHeader(http.StatusOK)
	}
}
