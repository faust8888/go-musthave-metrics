package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/faust8888/go-musthave-metrics/internal/hash"
	models "github.com/faust8888/go-musthave-metrics/internal/model"
	"github.com/faust8888/go-musthave-metrics/internal/retry"
)

// MetricsProvider supplies gauges and the accumulated PollCount for a report.
type MetricsProvider interface {
	Gauges() map[string]float64
	TakeAndResetPollCount() int64
}

// Sender posts metric batches to the server over HTTP.
type Sender struct {
	serverURL   string
	key         string
	client      *http.Client
	retryDelays []time.Duration
}

// NewSender returns a sender for serverURL. A non-empty key signs request bodies.
func NewSender(serverURL, key string) *Sender {
	return &Sender{
		serverURL:   serverURL,
		key:         key,
		client:      &http.Client{},
		retryDelays: retry.DefaultDelays,
	}
}

// NewSenderWithDelays is like NewSender but uses custom retry delays.
func NewSenderWithDelays(serverURL, key string, delays []time.Duration) *Sender {
	return &Sender{
		serverURL:   serverURL,
		key:         key,
		client:      &http.Client{},
		retryDelays: delays,
	}
}

// Send collects all current metrics and posts them in a single batch request,
// retrying on transient network errors.
func (s *Sender) Send(c MetricsProvider) error {
	return s.SendBatch(metricsFrom(c))
}

// SendBatch posts the given metrics in one gzip-compressed request.
func (s *Sender) SendBatch(metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}
	return s.postBatch(metrics)
}

func (s *Sender) postBatch(metrics []models.Metrics) error {
	// Build the compressed payload once; reuse across retries.
	// Hash is computed over the uncompressed JSON so it matches the server,
	// which verifies the body after gzip decompression.
	raw, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}
	payload, err := gzipBytes(raw)
	if err != nil {
		return err
	}
	return retry.Do(func() error {
		return s.doPost(payload, raw)
	}, isNetworkError, s.retryDelays)
}

func (s *Sender) doPost(payload, raw []byte) error {
	req, err := http.NewRequest(http.MethodPost, s.serverURL+"/updates/", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	if s.key != "" {
		req.Header.Set(hash.Header, hash.Sign(raw, s.key))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

var gzipWriterPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
		return w
	},
}

var gzipBufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func gzipBytes(raw []byte) ([]byte, error) {
	buf := gzipBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer gzipBufPool.Put(buf)

	gz := gzipWriterPool.Get().(*gzip.Writer)
	gz.Reset(buf)
	defer gzipWriterPool.Put(gz)

	if _, err := gz.Write(raw); err != nil {
		return nil, fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}

func isNetworkError(err error) bool {
	var urlErr *url.Error
	return errors.As(err, &urlErr)
}
