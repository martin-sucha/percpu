// Package percpu provides best-effort CPU-local sharded values.
package percpu

import (
	"golang.org/x/sys/cpu"
	"runtime"
	"sync/atomic"
	_ "unsafe"
)

// Values is a sharded set of values which have an affinity for a particular
// processor. This can be used to avoid cache contention when updating a shared
// value simultaneously from many goroutines.
//
// A zero value of a Values is ready to use.
// Values must not be copied after first use.
type Values[T any] struct {
	pad1 cpu.CacheLinePad // prevent false sharing

	// shards keeps the per-CPU pointers.
	// Grows in case GOMAXPROCS is increased.
	// Never shrinks.
	shards atomic.Pointer[[]*padded[T]]

	pad2 cpu.CacheLinePad // prevent false sharing
}

type padded[T any] struct {
	pad1 cpu.CacheLinePad // prevent false sharing
	v    T
	pad2 cpu.CacheLinePad // prevent false sharing
}

// Get returns a pointer to one of the values in v.
//
// The pointer tends to be the one associated with the current processor.
// However, goroutines can migrate at any time, and it may be the case
// that a different goroutine is accessing the same pointer concurrently.
// All access of the returned value must use further synchronization
// mechanisms.
//
// If a value for a given CPU does not exist yet, Values allocates a new zero value.
// The value is guaranteed to be allocated in a memory block
// with sufficient padding to avoid false sharing.
// Standard value alignment guarantees apply.
// This means that the implementation does NOT guarantee that a 64-bit
// integer will be aligned to the 64-bit boundary on 32-bit systems.
// See also Bugs section in the documentation of sync/atomic.
//
// A pointer returned by Get will be observed by Do forever,
// there isn't a way to free any of the values.
func (v *Values[T]) Get() *T {
	shardID := getProcID()

	shards := v.shards.Load()
	for shards == nil || shardID >= len(*shards) {
		// GOMAXPROCS has changed or shards was not initialized.
		newShardCount := runtime.GOMAXPROCS(0)
		if shardID >= newShardCount {
			// GOMAXPROCS might be lower than shardID+1 if GOMAXPROCS increased and then decreased.
			// Ensure we have enough space.
			newShardCount = shardID + 1
		}
		newShards := make([]*padded[T], newShardCount)
		nValid := 0
		if shards != nil {
			nValid = copy(newShards, *shards)
		}
		for i := nValid; i < newShardCount; i++ {
			newShards[i] = new(padded[T])
		}

		if v.shards.CompareAndSwap(shards, &newShards) {
			shards = &newShards
			break
		}
		// Another goroutine beat us, retry.
		shards = v.shards.Load()
	}

	slot := (*shards)[shardID]
	return &slot.v
}

// Do runs fn on all values in v.
//
// fn may be called zero or more times.
// fn might observe a new p before any goroutine calling Get has a chance to initialize it.
//
// The pointers might be concurrently used by other goroutines.
// The user is responsible for synchronizing access to p.
func (v *Values[T]) Do(fn func(p *T)) {
	shards := v.shards.Load()
	if shards == nil {
		return
	}

	for _, shard := range *shards {
		fn(&shard.v)
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
