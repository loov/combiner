package extcombiner

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// BoundedSleepy is a bounded non-spinning combiner queue
//
// Based on https://software.intel.com/en-us/blogs/2013/02/22/combineraggregator-synchronization-primitive
type BoundedSleepy struct {
	head    unsafe.Pointer // *boundedSleepyNode
	_       [7]uint64
	lock    sync.Mutex
	cond    sync.Cond
	_       [0]uint64
	batcher Batcher
	limit   int
}

type boundedSleepyNode struct {
	next     unsafe.Pointer // *boundedSleepyNode
	handoff  int64
	argument interface{}
}

// NewBoundedSleepy creates a BoundedSleepy queue.
func NewBoundedSleepy(batcher Batcher, limit int) *BoundedSleepy {
	c := &BoundedSleepy{
		batcher: batcher,
		limit:   limit,
		head:    nil,
	}
	c.cond.L = &c.lock
	return c
}

var boundedSleepyLockedElem = boundedSleepyNode{}
var boundedSleepyLockedNode = &boundedSleepyLockedElem
var boundedSleepyLocked = (unsafe.Pointer)(boundedSleepyLockedNode)

// Do passes value to Batcher and waits for completion
func (c *BoundedSleepy) Do(arg interface{}) {
	node := &boundedSleepyNode{argument: arg}

	var cmp unsafe.Pointer
	for {
		cmp = atomic.LoadPointer(&c.head)
		xchg := boundedSleepyLocked
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
		for node != boundedSleepyLockedNode {
			if count == c.limit {
				atomic.StoreInt64(&node.handoff, 1)
				c.batcher.Finish()

				c.lock.Lock()
				c.cond.Broadcast()
				c.lock.Unlock()
				return
			}
			next := (*boundedSleepyNode)(node.next)
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
			if cmp != boundedSleepyLocked {
				xchg = boundedSleepyLocked
			}
			if atomic.CompareAndSwapPointer(&c.head, cmp, xchg) {
				break
			}
		}

		// No more operations to combine, return.
		if cmp == boundedSleepyLocked {
			break
		}

		node = (*boundedSleepyNode)(cmp)

		// Execute the list of operations.
		for node != boundedSleepyLockedNode {
			if count == c.limit {
				atomic.StoreInt64(&node.handoff, 1)
				c.batcher.Finish()

				c.lock.Lock()
				c.cond.Broadcast()
				c.lock.Unlock()
				return
			}
			next := (*boundedSleepyNode)(node.next)
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
