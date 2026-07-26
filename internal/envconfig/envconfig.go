package envconfig

import (
	"fmt"
	"os"
	"strconv"
)

// String sets *dest to the value of the environment variable if it is non-empty.
func String(key string, dest *string) {
	if v := os.Getenv(key); v != "" {
		*dest = v
	}
}

// Int sets *dest to the parsed integer value of the environment variable.
// Returns an error if the variable is set but cannot be parsed as an integer.
func Int(key string, dest *int) error {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("invalid %s value %q: %w", key, v, err)
	}
	*dest = n
	return nil
}

// Bool sets *dest to the parsed boolean value of the environment variable.
// Returns an error if the variable is set but cannot be parsed as a boolean.
func Bool(key string, dest *bool) error {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fmt.Errorf("invalid %s value %q: %w", key, v, err)
	}
	*dest = b
	return nil
}
