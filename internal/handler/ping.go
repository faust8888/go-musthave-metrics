package handler

import (
	"context"
	"net/http"
)

// DBPinger is satisfied by *sql.DB and any test double.
type DBPinger interface {
	PingContext(ctx context.Context) error
}

// Ping returns a handler that checks the database connection.
// If db is nil the handler always returns 500.
func Ping(db DBPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			http.Error(w, "database not configured", http.StatusInternalServerError)
			return
		}
		if err := db.PingContext(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
