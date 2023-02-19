package percpu

import (
	"sync/atomic"
)

// A Counter is an int64 counter which may be efficiently incremented
// by many goroutines concurrently.
//
// The counter is sharded into several CPU-local values which are written and
// read independently. Thus, the Load and Reset methods do not observe a
// consistent view of the total if they are called concurrently to Add.
//
// For example, suppose goroutine G1 runs
//
//	counter.Add(1)
//	counter.Add(2)
//
// and, concurrently, G2 runs
//
//	t0 := counter.Reset()
//	// wait for G1 to finish executing
//	t1 := counter.Load()
//
// The value of t0 may be any of 0, 1, 2, or 3.
// The value of t1 may be any of 0, 1, 2, or 3 as well.
// However, t0+t1 must equal 3.
type Counter struct {
	vs Values[atomic.Int64]
}

// NewCounter returns a fresh Counter initialized to zero.
func NewCounter() *Counter {
	return &Counter{}
}

// Add adds n to the total count.
func (c *Counter) Add(n int64) {
	c.vs.Get().Add(n)
}

// Load computes the total counter value.
func (c *Counter) Load() int64 {
	var sum int64
	c.vs.Do(func(v *atomic.Int64) {
		sum += v.Load()
	})
	return sum
}

// Reset sets the counter to zero and reports the old value.
func (c *Counter) Reset() int64 {
	var sum int64
	c.vs.Do(func(v *atomic.Int64) {
		sum += v.Swap(0)
	})
	return sum
}
