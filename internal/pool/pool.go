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
	new   func() T
}

// New creates a pool that allocates missing objects with newFunc,
// the same role as sync.Pool.New.
func New[T Resetter](newFunc func() T) *Pool[T] {
	p := &Pool[T]{new: newFunc}
	p.inner.New = func() any {
		return newFunc()
	}
	return p
}

// Get returns an object from the pool, allocating one if the pool is empty.
func (p *Pool[T]) Get() T {
	v := p.inner.Get().(T)
	if isNil(v) {
		return p.new()
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
