package audit

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

// FileSink appends each event as a JSON line to an already-open file.
type FileSink struct {
	f  *os.File
	mu sync.Mutex
}

// NewFileSink opens path for append and keeps the descriptor until Close.
func NewFileSink(path string) (*FileSink, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &FileSink{f: f}, nil
}

// Write appends the event to the audit file.
func (s *FileSink) Write(event Event) {
	line, err := json.Marshal(event)
	if err != nil {
		log.Printf("audit file marshal: %v", err)
		return
	}
	line = append(line, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		return
	}
	if _, err = s.f.Write(line); err != nil {
		log.Printf("audit file write: %v", err)
	}
}

// Close closes the underlying file descriptor.
func (s *FileSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		return nil
	}
	err := s.f.Close()
	s.f = nil
	return err
}
