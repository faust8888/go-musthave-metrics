package handler

import "net/http"

// Auditor records successful metric-update requests.
type Auditor interface {
	Record(r *http.Request, metrics []string)
}

func recordAudit(auditor Auditor, r *http.Request, metrics []string) {
	if auditor == nil {
		return
	}
	auditor.Record(r, metrics)
}
