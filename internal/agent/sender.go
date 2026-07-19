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

func (s *Sender) Send(c MetricsProvider) error {
	for name, value := range c.Gauges() {
		v := value
		m := models.Metrics{ID: name, MType: models.Gauge, Value: &v}
		if err := s.postJSON(m); err != nil {
			return err
		}
	}

	pollCount := c.TakeAndResetPollCount()
	m := models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &pollCount}
	return s.postJSON(m)
}

func (s *Sender) postJSON(m models.Metrics) error {
	raw, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal metric: %w", err)
	}

	var buf bytes.Buffer
	gz, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if _, err = gz.Write(raw); err != nil {
		return fmt.Errorf("gzip write: %w", err)
	}
	if err = gz.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.serverURL+"/update", &buf)
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
