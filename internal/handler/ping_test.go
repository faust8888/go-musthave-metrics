package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
)

type mockPinger struct{ err error }

func (m *mockPinger) PingContext(_ context.Context) error { return m.err }

func TestPingHandler(t *testing.T) {
	tests := []struct {
		name     string
		db       handler.DBPinger
		wantCode int
	}{
		{"nil db returns 500", nil, http.StatusInternalServerError},
		{"healthy db returns 200", &mockPinger{}, http.StatusOK},
		{"unreachable db returns 500", &mockPinger{err: errors.New("connection refused")}, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()
			handler.Ping(tt.db)(w, req)
			if w.Code != tt.wantCode {
				t.Errorf("got status %d, want %d", w.Code, tt.wantCode)
			}
		})
	}
}
