package combiner

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

type Spinning struct {
	limit   int64
	batcher Batcher
	_       [5]uint64
	head    nodeptr
	_       [7]uint64
}

func NewSpinning(batcher Batcher, limit int) *Spinning {
	return &Spinning{limit: int64(limit), batcher: batcher}
}

//go:nosplit
//go:noinline
func (q *Spinning) Do(arg interface{}) {
	var mynode node
	my := &mynode
	my.argument = arg

	var cmp nodeptr
	for {
		cmp = atomic.LoadUintptr(&q.head)
		xchg := locked
		if cmp != 0 {
			xchg = my.ref()
			my.next = cmp
		}
		if atomic.CompareAndSwapUintptr(&q.head, cmp, xchg) {
			break
		}
	}

	handoff := false
	if cmp != 0 {
		// busy wait
		for i := 0; i < 8; i++ {
			next := atomic.LoadUintptr(&my.next)
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
			next := atomic.LoadUintptr(&my.next)
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
	defer q.batcher.Finish()

	q.batcher.Include(my.argument)
	count := int64(1)

	if handoff {
		goto combine
	}

combinecheck:
	for {
		cmp = atomic.LoadUintptr(&q.head)
		var xchg uintptr = 0
		if cmp != locked {
			xchg = locked
		}

		if atomic.CompareAndSwapUintptr(&q.head, cmp, xchg) {
			break
		}
	}

	// No more operations to combine, return.
	if cmp == locked {
		return
	}

combine:
	// Execute the list of operations.
	for cmp != locked {
		other := (*node)(unsafe.Pointer(cmp))
		if count == q.limit {
			atomic.StoreUintptr(&other.next, other.next|handoffTag)
			return
		}
		cmp = other.next

		q.batcher.Include(other.argument)
		count++
		// Mark completion.
		atomic.StoreUintptr(&other.next, 0)
	}

	goto combinecheck
}
