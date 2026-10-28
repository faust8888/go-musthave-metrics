package pool

import (
	"sync"
	"testing"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
)

type item struct {
	n int
	s string
	v []int
}

func (i *item) Reset() {
	if i == nil {
		return
	}
	i.n = 0
	i.s = ""
	i.v = i.v[:0]
}

func TestPool_GetAllocates(t *testing.T) {
	p := New[*item]()
	got := p.Get()
	if got == nil {
		t.Fatal("Get returned nil")
	}
}

func TestPool_PutResets(t *testing.T) {
	p := New[*item]()
	v := p.Get()
	v.n = 42
	v.s = "hello"
	v.v = append(v.v, 1, 2, 3)
	p.Put(v)

	got := p.Get()
	if got.n != 0 || got.s != "" || len(got.v) != 0 {
		t.Fatalf("object was not reset: %+v", got)
	}
}

func TestPool_PutNilIgnored(t *testing.T) {
	p := New[*item]()
	p.Put(nil)
	if p.Get() == nil {
		t.Fatal("Get after Put(nil) returned nil")
	}
}

func TestPool_Metrics(t *testing.T) {
	p := New[*models.Metrics]()
	m := p.Get()
	m.ID = "Alloc"
	m.MType = models.Gauge
	val := 1.5
	m.Value = &val
	p.Put(m)

	got := p.Get()
	if got.ID != "" || got.MType != "" || got.Hash != "" {
		t.Fatalf("metrics not reset: %+v", got)
	}
	if got.Value != nil && *got.Value != 0 {
		t.Fatalf("Value pointer: %+v", got.Value)
	}
}

func TestPool_Concurrent(t *testing.T) {
	p := New[*item]()
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				v := p.Get()
				v.n = j
				p.Put(v)
			}
		}()
	}
	wg.Wait()
}
