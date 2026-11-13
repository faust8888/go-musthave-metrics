package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/faust8888/go-musthave-metrics/internal/cryptokey"
)

// DecryptRSA decrypts request bodies marked with cryptokey.Header.
// Requests without a private key or without the header are passed through.
func DecryptRSA(priv *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if priv == nil || r.Header.Get(cryptokey.Header) != cryptokey.HeaderValue {
				next.ServeHTTP(w, r)
				return
			}
			body, err := io.ReadAll(r.Body)
			_ = r.Body.Close()
			if err != nil {
				http.Error(w, "failed to read encrypted body", http.StatusBadRequest)
				return
			}
			plain, err := cryptokey.Decrypt(priv, body)
			if err != nil {
				http.Error(w, "failed to decrypt request", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(plain))
			r.ContentLength = int64(len(plain))
			r.Header.Del(cryptokey.Header)
			next.ServeHTTP(w, r)
		})
	}
}
