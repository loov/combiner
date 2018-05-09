package extcombiner

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type Mutex struct {
	mu      sync.Mutex
	batcher Batcher
}

func NewMutex(batcher Batcher) *Mutex {
	c := &Mutex{}
	c.batcher = batcher
	return c
}

func (c *Mutex) Do(v interface{}) {
	c.mu.Lock()
	c.batcher.Start()
	c.batcher.Include(v)
	c.batcher.Finish()
	c.mu.Unlock()
}

type SpinMutex struct {
	mu      spinmutex
	batcher Batcher
}

func NewSpinMutex(batcher Batcher) *SpinMutex {
	c := &SpinMutex{}
	c.batcher = batcher
	return c
}

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
