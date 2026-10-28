package agent_test

import (
	"testing"

	"github.com/faust8888/go-musthave-metrics/internal/agent"
)

func BenchmarkCollector_Collect(b *testing.B) {
	c := agent.NewCollector()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Collect()
	}
}

func BenchmarkCollector_Snapshot(b *testing.B) {
	c := agent.NewCollector()
	c.Collect()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Snapshot()
	}
}

func BenchmarkCollector_CollectAndSnapshot(b *testing.B) {
	c := agent.NewCollector()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Collect()
		_ = c.Snapshot()
	}
}
