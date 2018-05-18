package combiner

import (
	"runtime"
)

// Spinning is a combiner queue with spinning waiters.
//
// This implementation is useful when the batcher work is small
// and there are few goroutines concurrently calling Do. A good example
// would be a shared data-structure.
//
// If very high performance is required benchmark replacing Batcher
// and argument with concrete implementation.
type Spinning struct {
	limit   int64
	batcher Batcher
	_       [5]uint64
	head    nodeptr
	_       [7]uint64
}

// NewSpinning creates a spinning combiner with the given limit
func NewSpinning(batcher Batcher, limit int) *Spinning {
	q := &Spinning{}
	q.Init(batcher, limit)
	return q
}

// Init initializes a Spinning combiner.
// Note: NewSpinning does this automatically.
func (q *Spinning) Init(batcher Batcher, limit int) {
	if limit < 0 {
		panic("combiner limit must be positive")
	}
	q.batcher = batcher
	q.limit = int64(limit)
}

// Do passes arg safely to batcher and calls Start / Finish.
// The methods maybe called in a different goroutine.
//go:nosplit
//go:noinline
func (q *Spinning) Do(arg interface{}) {
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
		// yielding busy wait
		for {
			next := atomicLoadNodeptr(&my.next)
			if next == 0 {
				return
			}
			if next&handoffTag != 0 {
				my.next &^= handoffTag
				handoff = true
				goto combining
			}
			runtime.Gosched()
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
		return
	}

combine:
	// Execute the list of operations.
	for cmp != locked {
		other := nodeptrToNode(cmp)
		if count == q.limit {
			atomicStoreNodeptr(&other.next, other.next|handoffTag)
			q.batcher.Finish()
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
