package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/faust8888/go-musthave-metrics/internal/hash"
)

func TestHashSHA256_NoKey_Passthrough(t *testing.T) {
	h := HashSHA256("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"id":"x"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	if got := w.Header().Get(hash.Header); got != "" {
		t.Errorf("HashSHA256 header should be empty without a key, got %q", got)
	}
}

func TestHashSHA256_ValidRequest(t *testing.T) {
	const key = "secret"
	body := `{"id":"Alloc","type":"gauge","value":1}`

	h := HashSHA256(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ := io.ReadAll(r.Body)
		if string(got) != body {
			t.Errorf("handler body: got %q, want %q", got, body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set(hash.Header, hash.Sign([]byte(body), key))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	want := hash.Sign(w.Body.Bytes(), key)
	if got := w.Header().Get(hash.Header); got != want {
		t.Errorf("response HashSHA256: got %q, want %q", got, want)
	}
}

func TestHashSHA256_InvalidHash(t *testing.T) {
	const key = "secret"
	body := `{"id":"Alloc","type":"gauge","value":1}`

	h := HashSHA256(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called for an invalid hash")
	}))

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	req.Header.Set(hash.Header, "00")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", w.Code)
	}
}

func TestHashSHA256_MissingHashWithBody(t *testing.T) {
	h := HashSHA256("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called when hash is missing")
	}))

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"id":"x"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", w.Code)
	}
}

func TestHashSHA256_GzipRequestBody(t *testing.T) {
	const key = "secret"
	body := `{"id":"Alloc","type":"gauge","value":1}`

	// Same order as the server: decompress first, then verify the plain body.
	h := GzipDecompress(HashSHA256(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ := io.ReadAll(r.Body)
		if string(got) != body {
			t.Errorf("handler body: got %q, want %q", got, body)
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})))

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/update", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set(hash.Header, hash.Sign([]byte(body), key))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200, body %q", w.Code, w.Body.String())
	}
}

func TestHashSHA256_EmptyBodyWithoutHeader(t *testing.T) {
	h := HashSHA256("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	if got := w.Header().Get(hash.Header); got == "" {
		t.Error("expected HashSHA256 on the response when a key is set")
	}
}
