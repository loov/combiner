package combiner_test

import (
	"testing"

	"github.com/loov/combiner"
)

func BenchmarkSpinningInvokerUncontended(b *testing.B) {
	q := combiner.NewSpinning(combiner.Invoker{}, 256)
	for i := 0; i < b.N; i++ {
		q.Do(func() {})
	}
}

func BenchmarkSpinningInvokerContended(b *testing.B) {
	q := combiner.NewSpinning(combiner.Invoker{}, 256)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Do(func() {})
		}
	})
}

func BenchmarkSpinningIncludeFnUncontended(b *testing.B) {
	bat := combiner.IncludeFunc(func(arg interface{}) {})
	q := combiner.NewSpinning(bat, 256)
	for i := 0; i < b.N; i++ {
		q.Do(0)
	}
}

func BenchmarkSpinningIncludeFnContended(b *testing.B) {
	bat := combiner.IncludeFunc(func(arg interface{}) {})
	q := combiner.NewSpinning(bat, 256)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Do(0)
		}
	})
}

func BenchmarkParkingInvokerUncontended(b *testing.B) {
	q := combiner.NewParking(combiner.Invoker{}, 256)
	for i := 0; i < b.N; i++ {
		q.Do(func() {})
	}
}

func BenchmarkParkingInvokerContended(b *testing.B) {
	q := combiner.NewParking(combiner.Invoker{}, 256)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Do(func() {})
		}
	})
}

func BenchmarkParkingIncludeFnUncontended(b *testing.B) {
	bat := combiner.IncludeFunc(func(arg interface{}) {})
	q := combiner.NewParking(bat, 256)
	for i := 0; i < b.N; i++ {
		q.Do(0)
	}
}

func BenchmarkParkingIncludeFnContended(b *testing.B) {
	bat := combiner.IncludeFunc(func(arg interface{}) {})
	q := combiner.NewParking(bat, 256)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Do(0)
		}
	})
}
