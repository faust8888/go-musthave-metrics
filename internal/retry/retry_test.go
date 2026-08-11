package retry

import (
	"errors"
	"testing"
	"time"
)

var zeroDelays = []time.Duration{0, 0, 0}

var errRetriable = errors.New("retriable")
var errFatal = errors.New("fatal")

func isRetriable(err error) bool { return errors.Is(err, errRetriable) }

func TestDo_SuccessFirstAttempt(t *testing.T) {
	calls := 0
	err := Do(func() error { calls++; return nil }, isRetriable, zeroDelays)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("want 1 call, got %d", calls)
	}
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		if calls < 3 {
			return errRetriable
		}
		return nil
	}, isRetriable, zeroDelays)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("want 3 calls, got %d", calls)
	}
}

func TestDo_ExhaustedReturnsLastError(t *testing.T) {
	calls := 0
	err := Do(func() error { calls++; return errRetriable }, isRetriable, zeroDelays)
	if !errors.Is(err, errRetriable) {
		t.Fatalf("want errRetriable, got %v", err)
	}
	// 1 initial + 3 retries = 4 total
	if calls != 4 {
		t.Errorf("want 4 calls, got %d", calls)
	}
}

func TestDo_NonRetriableStopsImmediately(t *testing.T) {
	calls := 0
	err := Do(func() error { calls++; return errFatal }, isRetriable, zeroDelays)
	if !errors.Is(err, errFatal) {
		t.Fatalf("want errFatal, got %v", err)
	}
	if calls != 1 {
		t.Errorf("want 1 call (no retries), got %d", calls)
	}
}

func TestDo_DelaysAreRespected(t *testing.T) {
	delays := []time.Duration{10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond}
	calls := 0
	start := time.Now()
	Do(func() error { calls++; return errRetriable }, isRetriable, delays) //nolint:errcheck
	elapsed := time.Since(start)

	// 3 retries × 10 ms = at least 30 ms
	if elapsed < 30*time.Millisecond {
		t.Errorf("delays not respected: elapsed %v, want ≥ 30ms", elapsed)
	}
}
