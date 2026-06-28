package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func Update(store repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Path: /update/<type>/<name>/<value>
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/update/"), "/")

		if len(parts) < 2 || parts[1] == "" {
			http.Error(w, "metric name required", http.StatusNotFound)
			return
		}

		if len(parts) < 3 || parts[2] == "" {
			http.Error(w, "metric value required", http.StatusBadRequest)
			return
		}

		metricType, metricName, metricValue := parts[0], parts[1], parts[2]

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
