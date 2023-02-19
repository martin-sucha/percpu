// Package percpu provides best-effort CPU-local sharded values.
package percpu

import (
	"golang.org/x/sys/cpu"
	"runtime"
	_ "unsafe"
)

// Pointer is a sharded set of values which have an affinity for a particular
// processor. This can be used to avoid cache contention when updating a shared
// value simultaneously from many goroutines.
type Pointer[T any] struct {
	pad1   cpu.CacheLinePad // prevent false sharing
	shards []*T
	pad2   cpu.CacheLinePad // prevent false sharing
}

// NewPointer constructs a Values using the provided constructor function to
// create each shard value.
func NewPointer[T any](newVal func() *T) *Pointer[T] {
	shards := make([]*T, runtime.GOMAXPROCS(0))
	for i := range shards {
		shards[i] = newVal()
	}
	return &Pointer[T]{shards: shards}
}

// Get returns one of the pointers in p.
//
// The pointer tends to be the one associated with the current processor.
// However, goroutines can migrate at any time, and it may be the case
// that a different goroutine is accessing the same pointer concurrently.
// All access of the returned value must use further synchronization
// mechanisms.
//
// BUG(cespare): If GOMAXPROCS has changed since a Pointer was created with
// NewPointer, Get may panic.
func (p *Pointer[T]) Get() *T {
	return p.shards[getProcID()]
}

// Do runs fn on all pointers in p.
func (p *Pointer[T]) Do(fn func(p *T)) {
	for _, pv := range p.shards {
		fn(pv)
	}
}

//go:linkname runtime_procPin runtime.procPin
func runtime_procPin() int

//go:linkname runtime_procUnpin runtime.procUnpin
func runtime_procUnpin() int

func getProcID() int {
	pid := runtime_procPin()
	runtime_procUnpin()
	return pid
}
