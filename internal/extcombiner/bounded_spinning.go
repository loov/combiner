package extcombiner

import (
	"sync/atomic"
	"unsafe"
)

// BoundedSpinning is a bounded spinning combiner queue
//
// Based on https://software.intel.com/en-us/blogs/2013/02/22/combineraggregator-synchronization-primitive
type BoundedSpinning struct {
	head    unsafe.Pointer // *boundedSpinningNode
	_       [7]uint64
	batcher Batcher
	limit   int
}

type boundedSpinningNode struct {
	next     unsafe.Pointer // *boundedSpinningNode
	handoff  int64
	argument interface{}
}

// NewBoundedSpinning creates a BoundedSpinning queue.
func NewBoundedSpinning(batcher Batcher, limit int) *BoundedSpinning {
	return &BoundedSpinning{
		batcher: batcher,
		limit:   limit,
		head:    nil,
	}
}

var boundedSpinningLockedElem = boundedSpinningNode{}
var boundedSpinningLockedNode = &boundedSpinningLockedElem
var boundedSpinningLocked = (unsafe.Pointer)(boundedSpinningLockedNode)

// Do passes value to Batcher and waits for completion
func (c *BoundedSpinning) Do(arg interface{}) {
	node := &boundedSpinningNode{argument: arg}

	var cmp unsafe.Pointer
	for {
		cmp = atomic.LoadPointer(&c.head)
		xchg := boundedSpinningLocked
		if cmp != nil {
			// There is already a combiner, enqueue itself.
			xchg = (unsafe.Pointer)(node)
			node.next = cmp
		}

		if atomic.CompareAndSwapPointer(&c.head, cmp, xchg) {
			break
		}
	}

	handoff := false
	if cmp != nil {
		// 2. If we are not the combiner, wait for arg.next to become nil
		// (which means the operation is finished).
		for try := 0; ; spin(&try) {
			next := atomic.LoadPointer(&node.next)
			if next == nil {
				return
			}

			if atomic.LoadInt64(&node.handoff) == 1 {
				// start combining from the current position
				handoff = true
				break
			}
		}
	}

	// 3. We are the combiner.

	// First, execute own operation.
	c.batcher.Start()
	defer c.batcher.Finish()

	var count int
	if !handoff {
		c.batcher.Do(node.argument)
		count++
	} else {
		// Execute the list of operations.
		for node != boundedSpinningLockedNode {
			if count == c.limit {
				atomic.StoreInt64(&node.handoff, 1)
				return
			}
			next := (*boundedSpinningNode)(node.next)
			c.batcher.Do(node.argument)
			count++
			// Mark completion.
			atomic.StorePointer(&node.next, nil)
			node = next
		}
	}

	// Then, look for combining opportunities.
	for {
		for {
			cmp = atomic.LoadPointer(&c.head)
			// If there are some operations in the list,
			// grab the list and replace with LOCKED.
			// Otherwise, exchange to nil.
			var xchg unsafe.Pointer = nil
			if cmp != boundedSpinningLocked {
				xchg = boundedSpinningLocked
			}
			if atomic.CompareAndSwapPointer(&c.head, cmp, xchg) {
				break
			}
		}

		// No more operations to combine, return.
		if cmp == boundedSpinningLocked {
			break
		}

		node = (*boundedSpinningNode)(cmp)

		// Execute the list of operations.
		for node != boundedSpinningLockedNode {
			if count == c.limit {
				atomic.StoreInt64(&node.handoff, 1)
				return
			}
			next := (*boundedSpinningNode)(node.next)
			c.batcher.Do(node.argument)
			count++
			// Mark completion.
			atomic.StorePointer(&node.next, nil)
			node = next
		}
	}
}
