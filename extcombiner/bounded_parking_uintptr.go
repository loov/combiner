package extcombiner

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// BoundedParkingUintptr is a bounded non-spinning combiner queue using uintptr internally
//
// Based on https://software.intel.com/en-us/blogs/2013/02/22/combineraggregator-synchronization-primitive
type BoundedParkingUintptr struct {
	head    uintptr // *boundedParkingUintptrNode
	_       [7]uint64
	lock    sync.Mutex
	cond    sync.Cond
	_       [0]uint64
	batcher Batcher
	limit   int
}

type boundedParkingUintptrNode struct {
	next     uintptr // *boundedParkingUintptrNode
	argument interface{}
}

// NewBoundedParkingUintptr creates a BoundedParkingUintptr queue.
func NewBoundedParkingUintptr(batcher Batcher, limit int) *BoundedParkingUintptr {
	c := &BoundedParkingUintptr{
		batcher: batcher,
		limit:   limit,
		head:    0,
	}
	c.cond.L = &c.lock
	return c
}

const (
	boundedParkingUintptrLocked     = uintptr(1)
	boundedParkingUintptrHandoffTag = uintptr(2)
)

// Do passes value to Batcher and waits for completion
func (c *BoundedParkingUintptr) Do(arg interface{}) {
	node := &boundedParkingUintptrNode{argument: arg}

	var cmp uintptr
	for {
		cmp = atomic.LoadUintptr(&c.head)
		xchg := boundedParkingUintptrLocked
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
		c.lock.Lock()
		for {
			next := atomic.LoadUintptr(&node.next)
			if next == 0 {
				c.lock.Unlock()
				return
			}

			if next&boundedParkingUintptrHandoffTag != 0 {
				node.next &^= boundedParkingUintptrHandoffTag
				// DO COMBINING
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
	c.batcher.Do(node.argument)
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
			if cmp != boundedParkingUintptrLocked {
				xchg = boundedParkingUintptrLocked
			}

			if atomic.CompareAndSwapUintptr(&c.head, cmp, xchg) {
				break
			}
		}

		// No more operations to combine, return.
		if cmp == boundedParkingUintptrLocked {
			break
		}

	combiner:
		// Execute the list of operations.
		for cmp != boundedParkingUintptrLocked {
			node = (*boundedParkingUintptrNode)(unsafe.Pointer(cmp))
			if count == c.limit {
				atomic.StoreUintptr(&node.next, node.next|boundedParkingUintptrHandoffTag)
				c.batcher.Finish()

				c.lock.Lock()
				c.cond.Broadcast()
				c.lock.Unlock()
				return
			}
			cmp = node.next

			c.batcher.Do(node.argument)
			count++
			// Mark completion.
			atomic.StoreUintptr(&node.next, 0)
		}
	}

	c.batcher.Finish()

	c.lock.Lock()
	c.cond.Broadcast()
	c.lock.Unlock()
}
