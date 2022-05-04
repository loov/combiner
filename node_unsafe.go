package combiner

import (
	"sync/atomic"
	"unsafe"
)

type nodeptr = uintptr

type node[T any] struct {
	next     nodeptr // *next
	argument T
}

func (n *node[T]) ref() nodeptr { return (nodeptr)(unsafe.Pointer(n)) }

const (
	locked     = nodeptr(1)
	handoffTag = nodeptr(2)
)

func atomicLoadNodeptr(p *nodeptr) nodeptr {
	return atomic.LoadUintptr(p)
}
func atomicStoreNodeptr(p *nodeptr, v nodeptr) {
	atomic.StoreUintptr(p, v)
}

func atomicCompareAndSwapNodeptr(addr *uintptr, old, new uintptr) bool {
	return atomic.CompareAndSwapUintptr(addr, old, new)
}

func nodeptrToNode[T any](p nodeptr) *node[T] { return (*node[T])(unsafe.Pointer(p)) }
