package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var compressibleTypes = map[string]bool{
	"application/json": true,
	"text/html":        true,
}

var gzipWriterPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
		return w
	},
}

var gzipReaderPool = sync.Pool{
	New: func() any {
		return new(gzip.Reader)
	},
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

func (grw *gzipResponseWriter) ensureWriter() {
	if grw.gz != nil {
		return
	}
	gz := gzipWriterPool.Get().(*gzip.Writer)
	gz.Reset(grw.ResponseWriter)
	grw.gz = gz
}

func (grw *gzipResponseWriter) WriteHeader(statusCode int) {
	grw.decide(nil)
	grw.ResponseWriter.WriteHeader(statusCode)
}

func (grw *gzipResponseWriter) Write(b []byte) (int, error) {
	grw.decide(b)
	if grw.useGzip {
		grw.ensureWriter()
		return grw.gz.Write(b)
	}
	return grw.ResponseWriter.Write(b)
}

func (grw *gzipResponseWriter) close() {
	if grw.gz == nil {
		if grw.useGzip {
			grw.ensureWriter()
		} else {
			return
		}
	}
	_ = grw.gz.Close()
	gzipWriterPool.Put(grw.gz)
	grw.gz = nil
}

// GzipCompress compresses responses for clients that send Accept-Encoding: gzip.
// Only application/json and text/html are compressed.
func GzipCompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		grw := &gzipResponseWriter{ResponseWriter: w}
		next.ServeHTTP(grw, r)
		grw.close()
	})
}

// GzipDecompress decompresses request bodies sent with Content-Encoding: gzip.
func GzipDecompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") != "gzip" {
			next.ServeHTTP(w, r)
			return
		}
		gr := gzipReaderPool.Get().(*gzip.Reader)
		if err := gr.Reset(r.Body); err != nil {
			gzipReaderPool.Put(gr)
			http.Error(w, "invalid gzip body", http.StatusBadRequest)
			return
		}
		defer func() {
			_ = gr.Close()
			gzipReaderPool.Put(gr)
		}()
		r.Body = io.NopCloser(gr)
		next.ServeHTTP(w, r)
	})
}
