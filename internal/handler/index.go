package handler

import (
	"bytes"
	"html"
	"net/http"
	"sort"
	"strconv"

	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

// Index handles GET / and renders an HTML page with all stored metrics.
func Index(store repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		gauges := store.GetAllGauges(ctx)
		counters := store.GetAllCounters(ctx)

		var buf bytes.Buffer
		buf.Grow(256 + 48*(len(gauges)+len(counters)))
		buf.WriteString("<html><head><title>Metrics</title></head><body><h1>Metrics</h1><ul>")

		gaugeNames := make([]string, 0, len(gauges))
		for name := range gauges {
			gaugeNames = append(gaugeNames, name)
		}
		sort.Strings(gaugeNames)
		for _, name := range gaugeNames {
			buf.WriteString("<li>")
			buf.WriteString(html.EscapeString(name))
			buf.WriteString(" (gauge) = ")
			buf.WriteString(strconv.FormatFloat(gauges[name], 'f', -1, 64))
			buf.WriteString("</li>")
		}

		counterNames := make([]string, 0, len(counters))
		for name := range counters {
			counterNames = append(counterNames, name)
		}
		sort.Strings(counterNames)
		for _, name := range counterNames {
			buf.WriteString("<li>")
			buf.WriteString(html.EscapeString(name))
			buf.WriteString(" (counter) = ")
			buf.WriteString(strconv.FormatInt(counters[name], 10))
			buf.WriteString("</li>")
		}

		buf.WriteString("</ul></body></html>")

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}
