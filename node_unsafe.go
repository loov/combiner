package combiner

import (
	"unsafe"
)

type nodeptr = uintptr

type node struct {
	next     nodeptr // *next
	argument interface{}
}

func (n *node) ref() nodeptr { return (nodeptr)(unsafe.Pointer(n)) }

const (
	locked     = nodeptr(1)
	handoffTag = nodeptr(2)
)
