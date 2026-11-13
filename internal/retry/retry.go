// Package retry retries operations with a fixed sequence of backoff delays.
package retry

import (
	"context"
	"time"
)

// DefaultDelays holds the wait intervals between successive retry attempts.
var DefaultDelays = []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

// Do calls fn once and retries up to len(delays) more times whenever
// shouldRetry(err) is true, sleeping delays[i] before each retry.
// Returns the error from the last attempt.
func Do(fn func() error, shouldRetry func(error) bool, delays []time.Duration) error {
	return DoContext(context.Background(), fn, shouldRetry, delays)
}

// DoContext is like Do but stops waiting or retrying when ctx is done.
func DoContext(ctx context.Context, fn func() error, shouldRetry func(error) bool, delays []time.Duration) error {
	err := fn()
	for _, d := range delays {
		if err == nil || !shouldRetry(err) {
			return err
		}
		if ctx.Err() != nil {
			return err
		}
		if d > 0 {
			timer := time.NewTimer(d)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return err
			case <-timer.C:
			}
		}
		if ctx.Err() != nil {
			return err
		}
		err = fn()
	}
	return err
}
