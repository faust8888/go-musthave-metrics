package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/faust8888/go-musthave-metrics/internal/hash"
)

// HashSHA256 verifies HashSHA256 when that request header is present and
// signs the response when key is non-empty. Requests without a key or
// without the header are passed through; a mismatch returns 400.
func HashSHA256(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}
			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))

			hw := &hashResponseWriter{ResponseWriter: w, key: key}
			defer hw.flush()

			got := r.Header.Get(hash.Header)
			if got != "" && !hash.Equal(got, body, key) {
				http.Error(hw, "invalid hash", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(hw, r)
		})
	}
}

type hashResponseWriter struct {
	http.ResponseWriter
	key     string
	buf     bytes.Buffer
	status  int
	flushed bool
}

func (hw *hashResponseWriter) WriteHeader(statusCode int) {
	hw.status = statusCode
}

func (hw *hashResponseWriter) Write(b []byte) (int, error) {
	return hw.buf.Write(b)
}

func (hw *hashResponseWriter) flush() {
	if hw.flushed {
		return
	}
	hw.flushed = true
	hw.Header().Set(hash.Header, hash.Sign(hw.buf.Bytes(), hw.key))
	if hw.status == 0 {
		hw.status = http.StatusOK
	}
	hw.ResponseWriter.WriteHeader(hw.status)
	if hw.buf.Len() > 0 {
		_, _ = hw.ResponseWriter.Write(hw.buf.Bytes())
	}
}
