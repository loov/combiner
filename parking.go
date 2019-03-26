package combiner

import (
	"runtime"
	"sync"
)

// Parking is a bounded non-spinning combiner queue.
//
// This implementation is useful when the batcher work is large
// ore there are many goroutines concurrently calling Do. A good example
// would be a appending to a file.
type Parking struct {
	limit   int64
	batcher Batcher
	_       [5]int64
	head    nodeptr
	_       [7]int64
	lock    sync.Mutex
	cond    sync.Cond
}

// NewParking creates a Parking combiner queue
func NewParking(batcher Batcher, limit int) *Parking {
	q := &Parking{}
	q.Init(batcher, limit)
	return q
}

// Init initializes a Parking combiner.
// Note: NewParking does this automatically.
func (q *Parking) Init(batcher Batcher, limit int) {
	if limit < 0 {
		panic("combiner limit must be positive")
	}

	q.batcher = batcher
	q.limit = int64(limit)
	q.cond.L = &q.lock
}

// Do passes value to Batcher and waits for completion
//go:nosplit
//go:noinline
func (q *Parking) Do(arg interface{}) {
	var mynode node
	my := &mynode
	my.argument = arg
	defer runtime.KeepAlive(my)

	var cmp nodeptr
	for {
		cmp = atomicLoadNodeptr(&q.head)
		xchg := locked
		if cmp != 0 {
			xchg = my.ref()
			my.next = cmp
		}
		if atomicCompareAndSwapNodeptr(&q.head, cmp, xchg) {
			break
		}
	}

	handoff := false
	if cmp != 0 {
		// busy wait
		for i := 0; i < 8; i++ {
			next := atomicLoadNodeptr(&my.next)
			if next == 0 {
				return
			}
			if next&handoffTag != 0 {
				my.next &^= handoffTag
				handoff = true
				goto combining
			}
		}

		q.lock.Lock()
		for {
			next := atomicLoadNodeptr(&my.next)
			if next == 0 {
				q.lock.Unlock()
				return
			}
			if next&handoffTag != 0 {
				my.next &^= handoffTag
				handoff = true
				q.lock.Unlock()
				goto combining
			}

			q.cond.Wait()
		}
	}

combining:
	q.batcher.Start()
	q.batcher.Do(my.argument)
	count := int64(1)

	if handoff {
		goto combine
	}

combinecheck:
	for {
		cmp = atomicLoadNodeptr(&q.head)
		var xchg uintptr = 0
		if cmp != locked {
			xchg = locked
		}

		if atomicCompareAndSwapNodeptr(&q.head, cmp, xchg) {
			break
		}
	}

	// No more operations to combine, return.
	if cmp == locked {
		q.batcher.Finish()

		q.lock.Lock()
		q.cond.Broadcast()
		q.lock.Unlock()
		return
	}

combine:
	// Execute the list of operations.
	for cmp != locked {
		other := nodeptrToNode(cmp)
		if count == q.limit {
			atomicStoreNodeptr(&other.next, other.next|handoffTag)

			q.batcher.Finish()

			q.lock.Lock()
			q.cond.Broadcast()
			q.lock.Unlock()
			return
		}
		cmp = other.next

		q.batcher.Do(other.argument)
		count++
		// Mark completion.
		atomicStoreNodeptr(&other.next, 0)
	}

	goto combinecheck
}
