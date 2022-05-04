package testsuite

import (
	"sync"
	"testing"
)

func RunBenchmarks(b *testing.B, setup *Setup) {
	b.Helper()
	setup.Bench(b, "Sum", benchSum)
}

func benchSum(b *testing.B, setup *Setup) {
	const N = 100

	_, combiner := setup.Make()
	defer StartClose(combiner)()

	b.ResetTimer()

	var wg sync.WaitGroup
	wg.Add(setup.Procs)

	left := b.N
	for i := 0; i < setup.Procs; i++ {
		chunk := left / (setup.Procs - i)
		go func(n int) {
			for i := 0; i < n; i++ {
				combiner.Do(int64(1))
			}
			wg.Done()
		}(chunk)
		left -= chunk
	}

	wg.Wait()
}
