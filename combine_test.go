package combiner_test

import (
	"sync"
	"testing"

	"loov.dev/combiner"
)

type Nop struct{}

func (n *Nop) Start()             {}
func (n *Nop) Do(arg interface{}) {}
func (n *Nop) Finish()            {}

func BenchmarkSpinningNopUncontended(b *testing.B) {
	var q combiner.Spinning
	q.Init(&Nop{}, 256)
	for i := 0; i < b.N; i++ {
		q.Do(nil)
	}
}

func BenchmarkSpinningNopContended(b *testing.B) {
	var q combiner.Spinning
	q.Init(&Nop{}, 256)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Do(nil)
		}
	})
}
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
func BenchmarkParkingNopUncontended(b *testing.B) {
	var q combiner.Parking
	q.Init(&Nop{}, 256)
	for i := 0; i < b.N; i++ {
		q.Do(nil)
	}
}

func BenchmarkParkingNopContended(b *testing.B) {
	var q combiner.Parking
	q.Init(&Nop{}, 256)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Do(nil)
		}
	})
}
