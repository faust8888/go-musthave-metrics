package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGzipCompress_JSON(t *testing.T) {
	h := GzipCompress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding: got %q", w.Header().Get("Content-Encoding"))
	}
	gr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	got, err := io.ReadAll(gr)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"ok":true}` {
		t.Errorf("body: %s", got)
	}
}

func TestGzipCompress_WithoutAccept(t *testing.T) {
	h := GzipCompress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("must not gzip without Accept-Encoding")
	}
	if w.Body.String() != `{"ok":true}` {
		t.Errorf("body: %s", w.Body.String())
	}
}

func TestGzipDecompress(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, _ = gw.Write([]byte(`{"id":"Alloc"}`))
	_ = gw.Close()

	var got []byte
	h := GzipDecompress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if string(got) != `{"id":"Alloc"}` {
		t.Errorf("decompressed: %s", got)
	}
}

func TestGzipDecompress_Invalid(t *testing.T) {
	h := GzipDecompress(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not run")
	}))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("not gzip")))
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}
