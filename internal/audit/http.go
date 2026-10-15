package audit

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// HTTPObserver POSTs each event as JSON to a remote URL.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver posts audit events as JSON to url.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Update POSTs the event to the configured URL.
func (o *HTTPObserver) Update(event Event) {
	body, err := json.Marshal(event)
	if err != nil {
		log.Printf("audit http marshal: %v", err)
		return
	}
	resp, err := o.client.Post(o.url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("audit http post: %v", err)
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}
