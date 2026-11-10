package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// DurationSec is a JSON duration in seconds. It accepts a duration string
// ("1s", "10s") or an integer number of seconds.
type DurationSec int

// UnmarshalJSON implements json.Unmarshaler.
func (d *DurationSec) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		parsed, err := time.ParseDuration(s)
		if err != nil {
			n, convErr := strconv.Atoi(s)
			if convErr != nil {
				return fmt.Errorf("config: duration %q: %w", s, err)
			}
			*d = DurationSec(n)
			return nil
		}
		*d = DurationSec(parsed / time.Second)
		return nil
	}
	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("config: duration: %w", err)
	}
	*d = DurationSec(n)
	return nil
}

// IntPtr returns a pointer to the duration as int seconds, or nil.
func (d *DurationSec) IntPtr() *int {
	if d == nil {
		return nil
	}
	v := int(*d)
	return &v
}
