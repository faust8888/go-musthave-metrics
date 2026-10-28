package audit

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// Event is a single audit record emitted after metrics were accepted.
type Event struct {
	// TS is the Unix timestamp when the metrics were accepted.
	TS int64 `json:"ts"`
	// Metrics lists the names of the stored metrics.
	Metrics []string `json:"metrics"`
	// IPAddress is the client IP that sent the request.
	IPAddress string `json:"ip_address"`
}

func newEvent(r *http.Request, metrics []string) Event {
	return Event{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: clientIP(r),
	}
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
