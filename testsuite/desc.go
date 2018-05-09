package testsuite

import (
	"strconv"
	"testing"
)

type Desc struct {
	Name    string
	Bounded bool
	Create  func(exe Batcher, bound int) Combiner
}

type Descs []Desc

type TestCase func(t *testing.T, procs int, create func(Batcher) Combiner)
type BenchCase func(b *testing.B, procs int, create func(Batcher) Combiner)

func (descs Descs) TestDefault(t *testing.T) {
	t.Helper()
	descs.Test(t, RunTests)
}
func (descs Descs) LatencyDefault(t *testing.T) {
	t.Helper()
	descs.Latency(t, RunLatency)
}
func (descs Descs) BenchmarkDefault(b *testing.B) {
	b.Helper()
	descs.Benchmark(b, RunBenchmarks)
}

func (descs Descs) Test(t *testing.T, test TestCase) {
	t.Helper()
	for _, desc := range descs {
		t.Run(desc.Name, func(t *testing.T) {
			t.Helper()
			bounds := TestBounds
			if !desc.Bounded {
				bounds = []int{0}
			}
			for _, procs := range TestProcs {
				for _, bound := range bounds {
					name := "p" + strconv.Itoa(procs) + "b" + strconv.Itoa(bound)
					t.Run(name, func(t *testing.T) {
						t.Helper()
						test(t, procs, func(exe Batcher) Combiner {
							return desc.Create(exe, bound)
						})
					})
				}
			}
		})
	}
}

func (descs Descs) Latency(t *testing.T, test TestCase) {
	t.Helper()
	for _, desc := range descs {
		t.Run(desc.Name, func(t *testing.T) {
			t.Helper()
			bounds := BenchBounds
			if !desc.Bounded {
				bounds = []int{0}
			}
			for _, procs := range BenchProcs {
				for _, bound := range bounds {
					name := "p" + strconv.Itoa(procs) + "b" + strconv.Itoa(bound)
					t.Run(name, func(t *testing.T) {
						t.Helper()
						test(t, procs, func(exe Batcher) Combiner {
							return desc.Create(exe, bound)
						})
					})
				}
			}
		})
	}
}

func (descs Descs) Benchmark(b *testing.B, bench BenchCase) {
	b.Helper()
	for _, desc := range descs {
		b.Run(desc.Name, func(b *testing.B) {
			b.Helper()
			bounds := BenchBounds
			if !desc.Bounded {
				bounds = []int{0}
			}
			for _, procs := range BenchProcs {
				for _, bound := range bounds {
					name := "p" + strconv.Itoa(procs) + "b" + strconv.Itoa(bound)
					b.Run(name, func(b *testing.B) {
						b.Helper()
						bench(b, procs, func(exe Batcher) Combiner {
							return desc.Create(exe, bound)
						})
					})
				}
			}
		})
	}
}
