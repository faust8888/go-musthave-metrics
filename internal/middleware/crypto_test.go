package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/faust8888/go-musthave-metrics/internal/cryptokey"
)

func TestDecryptRSA_PassthroughWithoutKey(t *testing.T) {
	h := DecryptRSA(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "plain" {
			t.Errorf("body: got %q", body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("plain")))
	h.ServeHTTP(httptest.NewRecorder(), req)
}

func TestDecryptRSA_DecryptsEnvelope(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte(`{"id":"Alloc"}`)
	enc, err := cryptokey.Encrypt(&priv.PublicKey, plain)
	if err != nil {
		t.Fatal(err)
	}

	h := DecryptRSA(priv)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !bytes.Equal(body, plain) {
			t.Errorf("decrypted: got %q, want %q", body, plain)
		}
		if r.Header.Get(cryptokey.Header) != "" {
			t.Error("encryption header must be stripped")
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(enc))
	req.Header.Set(cryptokey.Header, cryptokey.HeaderValue)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
}

func TestDecryptRSA_InvalidCiphertext(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	h := DecryptRSA(priv)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not run")
	}))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("not-encrypted")))
	req.Header.Set(cryptokey.Header, cryptokey.HeaderValue)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", w.Code)
	}
}
