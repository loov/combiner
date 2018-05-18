package combiner_test

import (
	"sync"
	"testing"

	"github.com/loov/combiner"
)

func BenchmarkSpinningInvokerUncontended(b *testing.B) {
	var q combiner.Spinning
	q.Init(combiner.Invoker{}, 256)
	for i := 0; i < b.N; i++ {
		q.Do(func() {})
	}
}

func BenchmarkSpinningInvokerContended(b *testing.B) {
	var q combiner.Spinning
	q.Init(combiner.Invoker{}, 256)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Do(func() {})
		}
	})
}
func BenchmarkLockInvokerUncontended(b *testing.B) {
	var mu sync.Mutex
	for i := 0; i < b.N; i++ {
		mu.Lock()
		(func() {})()
		mu.Unlock()
	}
}

func BenchmarkLockInvokerContended(b *testing.B) {
	var mu sync.Mutex
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			(func() {})()
			mu.Unlock()
		}
	})
}
func BenchmarkSpinningIncludeFnUncontended(b *testing.B) {
	bat := combiner.IncludeFunc(func(arg interface{}) {})
	var q combiner.Spinning
	q.Init(bat, 256)
	for i := 0; i < b.N; i++ {
		q.Do(0)
	}
}

func BenchmarkSpinningIncludeFnContended(b *testing.B) {
	bat := combiner.IncludeFunc(func(arg interface{}) {})
	var q combiner.Spinning
	q.Init(bat, 256)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Do(0)
		}
	})
}

func BenchmarkParkingInvokerUncontended(b *testing.B) {
	var q combiner.Parking
	q.Init(combiner.Invoker{}, 256)
	for i := 0; i < b.N; i++ {
		q.Do(func() {})
	}
}

func BenchmarkParkingInvokerContended(b *testing.B) {
	var q combiner.Parking
	q.Init(combiner.Invoker{}, 256)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Do(func() {})
		}
	})
}

func BenchmarkParkingIncludeFnUncontended(b *testing.B) {
	bat := combiner.IncludeFunc(func(arg interface{}) {})
	var q combiner.Parking
	q.Init(bat, 256)
	for i := 0; i < b.N; i++ {
		q.Do(0)
	}
}

func BenchmarkParkingIncludeFnContended(b *testing.B) {
	bat := combiner.IncludeFunc(func(arg interface{}) {})
	var q combiner.Parking
	q.Init(bat, 256)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Do(0)
		}
	})
}
