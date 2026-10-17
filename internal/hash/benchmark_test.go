package hash

import (
	"testing"
)

func BenchmarkSign(b *testing.B) {
	data := []byte(`[{"id":"Alloc","type":"gauge","value":123.45},{"id":"PollCount","type":"counter","delta":7}]`)
	key := "secret-key-for-benchmarks"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Sign(data, key)
	}
}
