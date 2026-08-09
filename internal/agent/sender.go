package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
)

type MetricsProvider interface {
	Gauges() map[string]float64
	TakeAndResetPollCount() int64
}

type Sender struct {
	serverURL string
	client    *http.Client
}

func NewSender(serverURL string) *Sender {
	return &Sender{
		serverURL: serverURL,
		client:    &http.Client{},
	}
}

// Send collects all current metrics and posts them in a single batch request.
func (s *Sender) Send(c MetricsProvider) error {
	gauges := c.Gauges()
	pollCount := c.TakeAndResetPollCount()

	metrics := make([]models.Metrics, 0, len(gauges)+1)
	for name, value := range gauges {
		v := value
		metrics = append(metrics, models.Metrics{ID: name, MType: models.Gauge, Value: &v})
	}
	metrics = append(metrics, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &pollCount})

	if len(metrics) == 0 {
		return nil
	}
	return s.postBatch(metrics)
}

func (s *Sender) postBatch(metrics []models.Metrics) error {
	raw, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return fmt.Errorf("gzip new writer: %w", err)
	}
	if _, err = gz.Write(raw); err != nil {
		return fmt.Errorf("gzip write: %w", err)
	}
	if err = gz.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.serverURL+"/updates/", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}
