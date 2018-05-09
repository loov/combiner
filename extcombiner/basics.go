package extcombiner

import (
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

// based on https://software.intel.com/en-us/blogs/2013/02/22/combineraggregator-synchronization-primitive
type BasicSleepy struct {
	head    unsafe.Pointer // *basicSleepyNode
	_       [7]uint64
	lock    sync.Mutex
	cond    sync.Cond
	_       [0]uint64
	batcher Batcher
}

type basicSleepyNode struct {
	argument interface{}
	next     unsafe.Pointer // *basicSleepyNode
}

func NewBasicSleepy(batcher Batcher) *BasicSleepy {
	c := &BasicSleepy{
		batcher: batcher,
		head:    nil,
	}
	c.cond.L = &c.lock
	return c
}

var basicSleepyLockedElem = basicSleepyNode{}
var basicSleepyLockedNode = &basicSleepyLockedElem
var basicSleepyLocked = (unsafe.Pointer)(basicSleepyLockedNode)

func (c *BasicSleepy) DoAsync(op interface{}) { c.do(op, true) }
func (c *BasicSleepy) Do(op interface{})      { c.do(op, false) }

func (c *BasicSleepy) do(op interface{}, async bool) {
	node := &basicSleepyNode{argument: op}

	var cmp unsafe.Pointer
	for {
		cmp = atomic.LoadPointer(&c.head)
		xchg := basicSleepyLocked
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
		c.batcher.Include(node.argument)

		for {
			for {
				cmp = atomic.LoadPointer(&c.head)
				var xchg unsafe.Pointer
				if cmp != basicSleepyLocked {
					xchg = basicSleepyLocked
				}

				if atomic.CompareAndSwapPointer(&c.head, cmp, xchg) {
					break
				}
			}

			if cmp == basicSleepyLocked {
				break
			}

			for cmp != basicSleepyLocked {
				node = (*basicSleepyNode)(unsafe.Pointer(cmp))
				cmp = node.next

				c.batcher.Include(node.argument)
				atomic.StorePointer(&node.next, nil)
			}
		}
		c.batcher.Finish()

		c.lock.Lock()
		c.cond.Broadcast()
		c.lock.Unlock()
	}
}
