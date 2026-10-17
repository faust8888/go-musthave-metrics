// Package audit records successful metric updates to a file and/or HTTP endpoint.
package audit

import (
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
)

const queueSize = 1024

// Sink receives audit events from Auditor.
type Sink interface {
	Write(Event)
}

type message struct {
	event Event
	flush chan struct{}
}

// Auditor records accepted metric updates and fans them out to sinks
// from a background worker so handlers stay off the I/O path.
//
// Record never blocks on sink I/O: the event is queued and dropped
// (with a log line) if the buffer is full.
type Auditor struct {
	mu     sync.RWMutex
	sinks  []Sink
	queue  chan message
	stop   chan struct{}
	done   chan struct{}
	closed atomic.Bool
}

// New starts a background worker and attaches file/HTTP sinks for every
// non-empty config value. The file sink is opened immediately so a bad
// path fails at startup. The returned Auditor must be closed on shutdown.
func New(filePath, url string) (*Auditor, error) {
	a := &Auditor{
		queue: make(chan message, queueSize),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	if filePath != "" {
		sink, err := NewFileSink(filePath)
		if err != nil {
			return nil, err
		}
		a.Add(sink)
	}
	if url != "" {
		a.Add(NewHTTPSink(url))
	}
	go a.loop()
	return a, nil
}

// Add registers a sink that will receive subsequent events.
func (a *Auditor) Add(s Sink) {
	if a == nil || s == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sinks = append(a.sinks, s)
}

// Notify delivers the event to every registered sink without holding the lock
// during I/O.
func (a *Auditor) Notify(event Event) {
	if a == nil {
		return
	}
	for _, s := range a.snapshot() {
		s.Write(event)
	}
}

// Record queues an audit event after metrics were stored successfully.
func (a *Auditor) Record(r *http.Request, metrics []string) {
	if a == nil || len(metrics) == 0 || a.closed.Load() {
		return
	}
	msg := message{event: newEvent(r, metrics)}
	select {
	case <-a.stop:
	case a.queue <- msg:
	default:
		log.Printf("audit queue full, dropping event")
	}
}

// Flush waits until every event queued so far has been delivered to sinks.
func (a *Auditor) Flush() {
	if a == nil || a.closed.Load() {
		return
	}
	done := make(chan struct{})
	select {
	case <-a.stop:
		return
	case a.queue <- message{flush: done}:
		<-done
	}
}

// Close stops the worker, drains the queue and closes file sinks.
func (a *Auditor) Close() error {
	if a == nil || !a.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(a.stop)
	<-a.done

	var first error
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, s := range a.sinks {
		if c, ok := s.(io.Closer); ok {
			if err := c.Close(); err != nil && first == nil {
				first = err
			}
		}
	}
	return first
}

func (a *Auditor) snapshot() []Sink {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if len(a.sinks) == 0 {
		return nil
	}
	out := make([]Sink, len(a.sinks))
	copy(out, a.sinks)
	return out
}

func (a *Auditor) loop() {
	defer close(a.done)
	for {
		select {
		case <-a.stop:
			a.drain()
			return
		case msg := <-a.queue:
			a.handle(msg)
		}
	}
}

func (a *Auditor) drain() {
	for {
		select {
		case msg := <-a.queue:
			a.handle(msg)
		default:
			return
		}
	}
}

func (a *Auditor) handle(msg message) {
	if msg.flush != nil {
		close(msg.flush)
		return
	}
	for _, s := range a.snapshot() {
		s.Write(msg.event)
	}
}
