package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
	"github.com/faust8888/go-musthave-metrics/internal/service"
)

// UpdatesBatch handles POST /updates/ and stores a JSON array of metrics.
func UpdatesBatch(store repository.Storage, auditor Auditor) http.HandlerFunc {
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
		if err := service.StoreMetrics(r.Context(), store, metrics); err != nil {
			var validationErr *service.ValidationError
			if errors.As(err, &validationErr) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		names := make([]string, 0, len(metrics))
		for _, m := range metrics {
			names = append(names, m.ID)
		}
		recordAudit(auditor, r, names)
		w.WriteHeader(http.StatusOK)
	}
}
