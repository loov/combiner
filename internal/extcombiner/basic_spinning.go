package extcombiner

import (
	"sync/atomic"
	"unsafe"
)

// BasicSpinning is an unbounded spinning combiner queue
//
// Based on https://software.intel.com/en-us/blogs/2013/02/22/combineraggregator-synchronization-primitive
type BasicSpinning struct {
	head    unsafe.Pointer // *basicSpinningNode
	_       [7]uint64
	batcher Batcher
}

type basicSpinningNode struct {
	next     unsafe.Pointer // *basicSpinningNode
	argument interface{}
}

// NewBasicSpinning creates a BasicSpinning queue.
func NewBasicSpinning(batcher Batcher) *BasicSpinning {
	return &BasicSpinning{
		batcher: batcher,
		head:    nil,
	}
}

var basicSpinningLockedElem = basicSpinningNode{}
var basicSpinningLockedNode = &basicSpinningLockedElem
var basicSpinningLocked = (unsafe.Pointer)(basicSpinningLockedNode)

// DoAsync passes value to Batcher without waiting for completion
func (c *BasicSpinning) DoAsync(op interface{}) { c.do(op, true) }

// Do passes value to Batcher and waits for completion
func (c *BasicSpinning) Do(op interface{}) { c.do(op, false) }

func (c *BasicSpinning) do(op interface{}, async bool) {
	node := &basicSpinningNode{argument: op}

	// c.head can be in 3 states:
	// c.head == 0: no operations in-flight, initial state.
	// c.head == LOCKED: single operation in-flight, no combining opportunities.
	// c.head == pointer to some basicSpinningNode that is subject to combining.
	//            The args are chainded into a lock-free list via 'next' fields.
	// node.next also can be in 3 states:
	// node.next == pointer to other node.
	// node.next == LOCKED: denotes the last node in the list.
	// node.next == 0: the operation is finished.

	// The function proceeds in 3 steps:
	// 1. If c.head == nil, exchange it to LOCKED and become the combiner.
	// Otherwise, enqueue own node into the c->head lock-free list.

	var cmp unsafe.Pointer
	for {
		cmp = atomic.LoadPointer(&c.head)
		xchg := basicSpinningLocked
		if cmp != nil {
			// There is already a combiner, enqueue itself.
			xchg = (unsafe.Pointer)(node)
			node.next = cmp
		}

		if atomic.CompareAndSwapPointer(&c.head, cmp, xchg) {
			break
		}
	}

	if cmp != nil {
		if async {
			return
		}

		// 2. If we are not the combiner, wait for node.next to become nil
		// (which means the operation is finished).
		for try := 0; atomic.LoadPointer(&node.next) != nil; spin(&try) {
		}
	} else {
		// 3. We are the combiner.

		// First, execute own operation.
		c.batcher.Start()
		defer c.batcher.Finish()

		c.batcher.Do(node.argument)

		// Then, look for combining opportunities.
		for {
			for {
				cmp = atomic.LoadPointer(&c.head)
				// If there are some operations in the list,
				// grab the list and replace with LOCKED.
				// Otherwise, exchange to nil.
				var xchg unsafe.Pointer
				if cmp != basicSpinningLocked {
					xchg = basicSpinningLocked
				}

				if atomic.CompareAndSwapPointer(&c.head, cmp, xchg) {
					break
				}
			}

			// No more operations to combine, return.
			if cmp == basicSpinningLocked {
				break
			}

			// Execute the list of operations.
			for cmp != basicSpinningLocked {
				node = (*basicSpinningNode)(unsafe.Pointer(cmp))
				cmp = node.next

				c.batcher.Do(node.argument)
				// Mark completion.
				atomic.StorePointer(&node.next, nil)
			}
		}
	}
}
