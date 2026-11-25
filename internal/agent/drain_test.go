package agent

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// blockUntilCancel hangs until the request context is cancelled.
type blockUntilCancel struct{}

func (blockUntilCancel) RoundTrip(req *http.Request) (*http.Response, error) {
	<-req.Context().Done()
	return nil, req.Context().Err()
}

func TestSendBatch_HonorsContext(t *testing.T) {
	s := NewSenderWithDelays("http://example.invalid", "", nil, nil)
	s.client.Transport = blockUntilCancel{}

	c := NewCollector()
	c.Collect()
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	start := time.Now()
	if err := s.SendBatch(ctx, c.Snapshot()); err == nil {
		t.Fatal("expected context deadline error")
	}
	if time.Since(start) > time.Second {
		t.Fatal("SendBatch ignored context cancellation")
	}
}

func TestRun_DrainDoesNotHang(t *testing.T) {
	a := New("http://example.invalid", "", 10*time.Millisecond, 20*time.Millisecond, 1, nil)
	a.drain = 150 * time.Millisecond
	a.sender.client.Timeout = 0
	a.sender.client.Transport = blockUntilCancel{}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.Run(ctx)
	}()

	time.Sleep(80 * time.Millisecond)
	start := time.Now()
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run hung after cancel")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("drain took %v, want well under 1s", elapsed)
	}
}
