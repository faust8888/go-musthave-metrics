// Package models defines the JSON metric payload used by the HTTP API.
package models

// Counter is the metric type that accumulates integer deltas.
const Counter = "counter"

// Gauge is the metric type that stores the latest floating-point value.
const Gauge = "gauge"

// Metrics is a single metric in JSON API requests and responses.
//
// Delta and Value are pointers so that a missing field can be distinguished
// from a zero value and omitted from JSON.
type Metrics struct {
	// ID is the metric name.
	ID string `json:"id"`
	// MType is the metric type: Counter or Gauge.
	MType string `json:"type"`
	// Delta is the counter increment; omitted for gauges.
	Delta *int64 `json:"delta,omitempty"`
	// Value is the gauge value; omitted for counters.
	Value *float64 `json:"value,omitempty"`
	// Hash is an optional per-metric signature from older API versions.
	Hash string `json:"hash,omitempty"`
}
