package retry

import "time"

// DefaultDelays holds the wait intervals between successive retry attempts.
var DefaultDelays = []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

// Do calls fn once and retries up to len(delays) more times whenever
// shouldRetry(err) is true, sleeping delays[i] before each retry.
// Returns the error from the last attempt.
func Do(fn func() error, shouldRetry func(error) bool, delays []time.Duration) error {
	err := fn()
	for _, d := range delays {
		if err == nil || !shouldRetry(err) {
			return err
		}
		time.Sleep(d)
		err = fn()
	}
	return err
}
