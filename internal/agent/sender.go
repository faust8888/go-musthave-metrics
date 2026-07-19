package agent

import (
	"bytes"
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
	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal metric: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.serverURL+"/update", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}
