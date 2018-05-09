package extcombiner

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// Mutex is a combiner queue that uses mutex to serialize operations.
//
// Not recommended, only for comparison.
type Mutex struct {
	mu      sync.Mutex
	batcher Batcher
}

// NewMutex creates a combiner queue based on a mutex.
func NewMutex(batcher Batcher) *Mutex {
	c := &Mutex{}
	c.batcher = batcher
	return c
}

// Do passes value to Batcher and waits for completion
func (c *Mutex) Do(v interface{}) {
	c.mu.Lock()
	c.batcher.Start()
	c.batcher.Include(v)
	c.batcher.Finish()
	c.mu.Unlock()
}

// SpinMutex is a combiner queue that uses a spinning mutex to serialize operations.
//
// Not recommended, only for comparison.
type SpinMutex struct {
	mu      spinmutex
	batcher Batcher
}

// NewSpinMutex creates a combiner queue based on a spin mutex.
func NewSpinMutex(batcher Batcher) *SpinMutex {
	c := &SpinMutex{}
	c.batcher = batcher
	return c
}

// Do passes value to Batcher and waits for completion
func (c *SpinMutex) Do(v interface{}) {
	c.mu.Lock()
	c.batcher.Start()
	c.batcher.Include(v)
	c.batcher.Finish()
	c.mu.Unlock()
}

type spinmutex struct {
	locked int64
	_      [7]int64
}

func (m *spinmutex) Lock() {
	for atomic.SwapInt64(&m.locked, 1) == 1 {
		for try := 0; atomic.LoadInt64(&m.locked) == 1; try++ {
			if try > 256 {
				runtime.Gosched()
			}
		}
	}
}

func (m *spinmutex) Unlock() {
	atomic.StoreInt64(&m.locked, 0)
}
