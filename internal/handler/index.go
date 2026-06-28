package handler

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func Index(store repository.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gauges := store.GetAllGauges()
		counters := store.GetAllCounters()

		var sb strings.Builder
		sb.WriteString("<html><head><title>Metrics</title></head><body><h1>Metrics</h1><ul>")

		gaugeNames := make([]string, 0, len(gauges))
		for name := range gauges {
			gaugeNames = append(gaugeNames, name)
		}
		sort.Strings(gaugeNames)
		for _, name := range gaugeNames {
			fmt.Fprintf(&sb, "<li>%s (gauge) = %s</li>", name, strconv.FormatFloat(gauges[name], 'f', -1, 64))
		}

		counterNames := make([]string, 0, len(counters))
		for name := range counters {
			counterNames = append(counterNames, name)
		}
		sort.Strings(counterNames)
		for _, name := range counterNames {
			fmt.Fprintf(&sb, "<li>%s (counter) = %d</li>", name, counters[name])
		}

		sb.WriteString("</ul></body></html>")

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, sb.String())
	}
}
