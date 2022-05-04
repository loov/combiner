package combiner_test

import (
	"sync"
	"testing"

	"loov.dev/combiner"
)

type Nop struct{}

func (n *Nop) Start()     {}
func (n *Nop) Do(arg int) {}
func (n *Nop) Finish()    {}

func BenchmarkLockNopUncontended(b *testing.B) {
	var mu sync.Mutex
	for i := 0; i < b.N; i++ {
		mu.Lock()
		{
			// intentionally empty section
		}
		mu.Unlock()
	}
}

func BenchmarkLockNopContended(b *testing.B) {
	var mu sync.Mutex
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			{
				// intentionally empty section
			}
			mu.Unlock()
		}
	})
}

func BenchmarkNopUncontended(b *testing.B) {
	var q combiner.Queue[int]
	q.Init(&Nop{}, 256)
	for i := 0; i < b.N; i++ {
		q.Do(123)
	}
}

func BenchmarkNopContended(b *testing.B) {
	var q combiner.Queue[int]
	q.Init(&Nop{}, 256)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Do(122)
		}
	})
}
