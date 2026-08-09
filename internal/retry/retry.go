package retry

import "time"

// Delays holds the wait intervals between successive retry attempts.
// Override in tests to avoid sleeping.
var Delays = []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

// Do calls fn once and retries up to len(Delays) more times whenever
// shouldRetry(err) is true, sleeping Delays[i] before each retry.
// Returns the error from the last attempt.
func Do(fn func() error, shouldRetry func(error) bool) error {
	err := fn()
	for _, d := range Delays {
		if err == nil || !shouldRetry(err) {
			return err
		}
		time.Sleep(d)
		err = fn()
	}
	return err
}
