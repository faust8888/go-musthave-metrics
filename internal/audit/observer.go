// Package audit implements the Observer pattern for metric-update audit events.
package audit

import (
	"net/http"
	"sync"
)

// Observer receives audit events (the observer side of the pattern).
type Observer interface {
	// Update is called when a new audit event is emitted.
	Update(Event)
}

// Subject keeps the list of observers and notifies them of new events.
type Subject struct {
	mu        sync.RWMutex
	observers []Observer
}

// New builds a subject and attaches file/HTTP observers for every non-empty config value.
func New(filePath, url string) *Subject {
	s := &Subject{}
	if filePath != "" {
		s.Attach(NewFileObserver(filePath))
	}
	if url != "" {
		s.Attach(NewHTTPObserver(url))
	}
	return s
}

// Attach registers an observer that will receive subsequent events.
func (s *Subject) Attach(o Observer) {
	if o == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observers = append(s.observers, o)
}

// Notify delivers the event to every attached observer.
func (s *Subject) Notify(event Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, o := range s.observers {
		o.Update(event)
	}
}

// Record emits an audit event after metrics were stored successfully.
func (s *Subject) Record(r *http.Request, metrics []string) {
	if s == nil || len(metrics) == 0 {
		return
	}
	s.mu.RLock()
	empty := len(s.observers) == 0
	s.mu.RUnlock()
	if empty {
		return
	}
	s.Notify(newEvent(r, metrics))
}
