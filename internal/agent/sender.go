package agent

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
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
	gauges := c.Gauges()
	for name, value := range gauges {
		url := fmt.Sprintf("%s/update/gauge/%s/%s",
			s.serverURL, name, strconv.FormatFloat(value, 'f', -1, 64))
		if err := s.post(url); err != nil {
			return err
		}
	}

	pollCount := c.TakeAndResetPollCount()
	url := fmt.Sprintf("%s/update/counter/PollCount/%d", s.serverURL, pollCount)
	if err := s.post(url); err != nil {
		return err
	}

	return nil
}

func (s *Sender) post(url string) error {
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}
