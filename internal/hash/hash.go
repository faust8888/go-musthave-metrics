package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Header is the HTTP header that carries the HMAC-SHA256 signature.
const Header = "HashSHA256"

// Sign returns the hex-encoded HMAC-SHA256 of data keyed with key.
func Sign(data []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// Equal reports whether got matches the HMAC-SHA256 of data keyed with key.
func Equal(got string, data []byte, key string) bool {
	return hmac.Equal([]byte(got), []byte(Sign(data, key)))
}
