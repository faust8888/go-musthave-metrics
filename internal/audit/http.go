package audit

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// HTTPSink POSTs each event as JSON to a remote URL.
type HTTPSink struct {
	url    string
	client *http.Client
}

// NewHTTPSink posts audit events as JSON to url.
func NewHTTPSink(url string) *HTTPSink {
	return &HTTPSink{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Write POSTs the event to the configured URL and logs a non-2xx status.
func (s *HTTPSink) Write(event Event) {
	body, err := json.Marshal(event)
	if err != nil {
		log.Printf("audit http marshal: %v", err)
		return
	}
	resp, err := s.client.Post(s.url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("audit http post: %v", err)
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("audit http status: %d", resp.StatusCode)
	}
}
