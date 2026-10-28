// Package pool provides a generic object pool for types with a Reset method.
package pool

import (
	"reflect"
	"sync"
)

// Resetter is implemented by values that can restore themselves to a blank state.
type Resetter interface {
	Reset()
}

// Pool reuses objects of a single type T. T must have Reset(); Put calls it
// before the object goes back into the pool.
type Pool[T Resetter] struct {
	inner sync.Pool
}

// New creates an empty pool for T.
func New[T Resetter]() *Pool[T] {
	p := &Pool[T]{}
	p.inner.New = func() any {
		return alloc[T]()
	}
	return p
}

// Get returns an object from the pool, allocating one if the pool is empty.
func (p *Pool[T]) Get() T {
	v := p.inner.Get().(T)
	if isNil(v) {
		return alloc[T]()
	}
	return v
}

// Put resets v and returns it to the pool. A nil v is ignored.
func (p *Pool[T]) Put(v T) {
	if isNil(v) {
		return
	}
	v.Reset()
	p.inner.Put(v)
}

func isNil[T any](v T) bool {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return rv.IsNil()
	}
	return false
}

func alloc[T any]() T {
	rt := reflect.TypeOf((*T)(nil)).Elem()
	if rt.Kind() == reflect.Ptr {
		return reflect.New(rt.Elem()).Interface().(T)
	}
	var v T
	return v
}
