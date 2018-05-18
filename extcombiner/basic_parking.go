package extcombiner

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// BasicParking is an unbounded non-spinning combiner queue
//
// Based on https://software.intel.com/en-us/blogs/2013/02/22/combineraggregator-synchronization-primitive
type BasicParking struct {
	head    unsafe.Pointer // *basicParkingNode
	_       [7]uint64
	lock    sync.Mutex
	cond    sync.Cond
	_       [0]uint64
	batcher Batcher
}

type basicParkingNode struct {
	argument interface{}
	next     unsafe.Pointer // *basicParkingNode
}

// NewBasicParking creates a BasicParking queue.
func NewBasicParking(batcher Batcher) *BasicParking {
	c := &BasicParking{
		batcher: batcher,
		head:    nil,
	}
	c.cond.L = &c.lock
	return c
}

var basicParkingLockedElem = basicParkingNode{}
var basicParkingLockedNode = &basicParkingLockedElem
var basicParkingLocked = (unsafe.Pointer)(basicParkingLockedNode)

// DoAsync passes value to Batcher without waiting for completion
func (c *BasicParking) DoAsync(op interface{}) { c.do(op, true) }

// Do passes value to Batcher and waits for completion
func (c *BasicParking) Do(op interface{}) { c.do(op, false) }

func (c *BasicParking) do(op interface{}, async bool) {
	node := &basicParkingNode{argument: op}

	var cmp unsafe.Pointer
	for {
		cmp = atomic.LoadPointer(&c.head)
		xchg := basicParkingLocked
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

		for try := 0; try < busyspin; try++ {
			if atomic.LoadPointer(&node.next) == nil {
				return
			}
			runtime.Gosched()
		}

		c.lock.Lock()
		for atomic.LoadPointer(&node.next) != nil {
			c.cond.Wait()
		}
		c.lock.Unlock()
	} else {
		c.batcher.Start()
		c.batcher.Do(node.argument)

		for {
			for {
				cmp = atomic.LoadPointer(&c.head)
				var xchg unsafe.Pointer
				if cmp != basicParkingLocked {
					xchg = basicParkingLocked
				}

				if atomic.CompareAndSwapPointer(&c.head, cmp, xchg) {
					break
				}
			}

			if cmp == basicParkingLocked {
				break
			}

			for cmp != basicParkingLocked {
				node = (*basicParkingNode)(unsafe.Pointer(cmp))
				cmp = node.next

				c.batcher.Do(node.argument)
				atomic.StorePointer(&node.next, nil)
			}
		}
		c.batcher.Finish()

		c.lock.Lock()
		c.cond.Broadcast()
		c.lock.Unlock()
	}
}
