package extcombiner

import (
	"sync/atomic"
	"unsafe"
)

// BoundedSpinningUintptr is a bounded spinning combiner queue using uintptr internally
//
// Based on https://software.intel.com/en-us/blogs/2013/02/22/combineraggregator-synchronization-primitive
type BoundedSpinningUintptr struct {
	head    uintptr // *boundedSpinningUintptrNode
	_       [7]uint64
	batcher Batcher
	limit   int
}

type boundedSpinningUintptrNode struct {
	next     uintptr // *boundedSpinningUintptrNode
	argument interface{}
}

// NewBoundedSpinningUintptr creates a BoundedSpinningUintptr queue.
func NewBoundedSpinningUintptr(batcher Batcher, limit int) *BoundedSpinningUintptr {
	return &BoundedSpinningUintptr{
		batcher: batcher,
		limit:   limit,
		head:    0,
	}
}

const (
	boundedSpinningUintptrLocked     = uintptr(1)
	boundedSpinningUintptrHandoffTag = uintptr(2)
)

// Do passes value to Batcher and waits for completion
func (c *BoundedSpinningUintptr) Do(arg interface{}) {
	node := &boundedSpinningUintptrNode{argument: arg}

	var cmp uintptr
	for {
		cmp = atomic.LoadUintptr(&c.head)
		xchg := boundedSpinningUintptrLocked
		if cmp != 0 {
			// There is already a combiner, enqueue itself.
			xchg = uintptr(unsafe.Pointer(node))
			node.next = cmp
		}

		if atomic.CompareAndSwapUintptr(&c.head, cmp, xchg) {
			break
		}
	}

	count := 0
	handoff := false
	if cmp != 0 {
		// 2. If we are not the combiner, wait for arg.next to become nil
		// (which means the operation is finished).
		for try := 0; ; spin(&try) {
			next := atomic.LoadUintptr(&node.next)
			if next == 0 {
				return
			}

			if next&boundedSpinningUintptrHandoffTag != 0 {
				node.next &^= boundedSpinningUintptrHandoffTag
				// DO COMBINING
				handoff = true
				break
			}
		}
	}

	// 3. We are the combiner.

	// First, execute own operation.
	c.batcher.Start()
	defer c.batcher.Finish()
	c.batcher.Include(node.argument)
	count++

	// Then, look for combining opportunities.
	for {
		if handoff { // using goto, to keep it similar to D. Vyukov-s implementation
			handoff = false
			goto combiner
		}

		for {
			cmp = atomic.LoadUintptr(&c.head)
			// If there are some operations in the list,
			// grab the list and replace with LOCKED.
			// Otherwise, exchange to nil.
			var xchg uintptr = 0
			if cmp != boundedSpinningUintptrLocked {
				xchg = boundedSpinningUintptrLocked
			}

			if atomic.CompareAndSwapUintptr(&c.head, cmp, xchg) {
				break
			}
		}

		// No more operations to combine, return.
		if cmp == boundedSpinningUintptrLocked {
			break
		}

	combiner:
		// Execute the list of operations.
		for cmp != boundedSpinningUintptrLocked {
			node = (*boundedSpinningUintptrNode)(unsafe.Pointer(cmp))
			if count == c.limit {
				atomic.StoreUintptr(&node.next, node.next|boundedSpinningUintptrHandoffTag)
				return
			}
			cmp = node.next

			c.batcher.Include(node.argument)
			count++
			// Mark completion.
			atomic.StoreUintptr(&node.next, 0)
		}
	}
}
