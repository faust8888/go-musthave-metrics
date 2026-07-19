package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

var compressibleTypes = map[string]bool{
	"application/json": true,
	"text/html":        true,
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz      *gzip.Writer
	useGzip bool
	decided bool
}

func (grw *gzipResponseWriter) decide(b []byte) {
	if grw.decided {
		return
	}
	grw.decided = true
	ct := grw.Header().Get("Content-Type")
	if ct == "" && len(b) > 0 {
		ct = http.DetectContentType(b)
	}
	if i := strings.IndexByte(ct, ';'); i != -1 {
		ct = strings.TrimSpace(ct[:i])
	}
	if compressibleTypes[ct] {
		grw.useGzip = true
		grw.Header().Set("Content-Encoding", "gzip")
		grw.Header().Del("Content-Length")
	}
}

func (grw *gzipResponseWriter) WriteHeader(statusCode int) {
	grw.decide(nil)
	grw.ResponseWriter.WriteHeader(statusCode)
}

func (grw *gzipResponseWriter) Write(b []byte) (int, error) {
	grw.decide(b)
	if grw.useGzip {
		return grw.gz.Write(b)
	}
	return grw.ResponseWriter.Write(b)
}

// GzipCompress compresses responses for clients that send Accept-Encoding: gzip.
// Only application/json and text/html are compressed.
func GzipCompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
		defer gz.Close()
		grw := &gzipResponseWriter{ResponseWriter: w, gz: gz}
		next.ServeHTTP(grw, r)
	})
}

// GzipDecompress decompresses request bodies sent with Content-Encoding: gzip.
func GzipDecompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			next.ServeHTTP(w, r)
			return
		}
		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "invalid gzip body", http.StatusBadRequest)
			return
		}
		defer gr.Close()
		r.Body = io.NopCloser(gr)
		next.ServeHTTP(w, r)
	})
}
