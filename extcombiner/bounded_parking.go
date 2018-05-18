package extcombiner

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// BoundedParking is a bounded non-spinning combiner queue
//
// Based on https://software.intel.com/en-us/blogs/2013/02/22/combineraggregator-synchronization-primitive
type BoundedParking struct {
	head    unsafe.Pointer // *boundedParkingNode
	_       [7]uint64
	lock    sync.Mutex
	cond    sync.Cond
	_       [0]uint64
	batcher Batcher
	limit   int
}

type boundedParkingNode struct {
	next     unsafe.Pointer // *boundedParkingNode
	handoff  int64
	argument interface{}
}

// NewBoundedParking creates a BoundedParking queue.
func NewBoundedParking(batcher Batcher, limit int) *BoundedParking {
	c := &BoundedParking{
		batcher: batcher,
		limit:   limit,
		head:    nil,
	}
	c.cond.L = &c.lock
	return c
}

var boundedParkingLockedElem = boundedParkingNode{}
var boundedParkingLockedNode = &boundedParkingLockedElem
var boundedParkingLocked = (unsafe.Pointer)(boundedParkingLockedNode)

// Do passes value to Batcher and waits for completion
func (c *BoundedParking) Do(arg interface{}) {
	node := &boundedParkingNode{argument: arg}

	var cmp unsafe.Pointer
	for {
		cmp = atomic.LoadPointer(&c.head)
		xchg := boundedParkingLocked
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
		c.lock.Lock()
		for {
			next := atomic.LoadPointer(&node.next)
			if next == nil {
				c.lock.Unlock()
				return
			}
			if atomic.LoadInt64(&node.handoff) == 1 {
				// start combining from the current position
				handoff = true
				break
			}
			c.cond.Wait()
		}
		c.lock.Unlock()
	}

	// 3. We are the combiner.

	// First, execute own operation.
	c.batcher.Start()

	var count int
	if !handoff {
		c.batcher.Include(node.argument)
		count++
	} else {
		// Execute the list of operations.
		for node != boundedParkingLockedNode {
			if count == c.limit {
				atomic.StoreInt64(&node.handoff, 1)
				c.batcher.Finish()

				c.lock.Lock()
				c.cond.Broadcast()
				c.lock.Unlock()
				return
			}
			next := (*boundedParkingNode)(node.next)
			c.batcher.Include(node.argument)
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
			if cmp != boundedParkingLocked {
				xchg = boundedParkingLocked
			}
			if atomic.CompareAndSwapPointer(&c.head, cmp, xchg) {
				break
			}
		}

		// No more operations to combine, return.
		if cmp == boundedParkingLocked {
			break
		}

		node = (*boundedParkingNode)(cmp)

		// Execute the list of operations.
		for node != boundedParkingLockedNode {
			if count == c.limit {
				atomic.StoreInt64(&node.handoff, 1)
				c.batcher.Finish()

				c.lock.Lock()
				c.cond.Broadcast()
				c.lock.Unlock()
				return
			}
			next := (*boundedParkingNode)(node.next)
			c.batcher.Include(node.argument)
			count++
			// Mark completion.
			atomic.StorePointer(&node.next, nil)
			node = next
		}
	}

	c.batcher.Finish()

	c.lock.Lock()
	c.cond.Broadcast()
	c.lock.Unlock()
}
