package audit

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

// FileObserver appends each event as a JSON line to a file.
type FileObserver struct {
	path string
	mu   sync.Mutex
}

// NewFileObserver writes audit events as JSON lines to path.
func NewFileObserver(path string) *FileObserver {
	return &FileObserver{path: path}
}

// Update appends the event to the audit file.
func (o *FileObserver) Update(event Event) {
	line, err := json.Marshal(event)
	if err != nil {
		log.Printf("audit file marshal: %v", err)
		return
	}
	line = append(line, '\n')

	o.mu.Lock()
	defer o.mu.Unlock()

	f, err := os.OpenFile(o.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("audit file open: %v", err)
		return
	}
	defer func() { _ = f.Close() }()
	if _, err = f.Write(line); err != nil {
		log.Printf("audit file write: %v", err)
	}
}
