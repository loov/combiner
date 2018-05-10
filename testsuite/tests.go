package testsuite

import (
	"sync"
	"testing"
)

func RunTests(t *testing.T, setup *Setup) {
	t.Helper()
	setup.Test(t, "Sum", testSum)
	setup.Test(t, "SumSequence", testSum)
}

func testSum(t *testing.T, setup *Setup) {
	const N = 100

	worker, combiner := setup.Make()
	defer StartClose(combiner)()

	var wg sync.WaitGroup

	wg.Add(setup.Procs)
	for proc := 0; proc < setup.Procs; proc++ {
		go func() {
			for i := int64(0); i < N; i++ {
				combiner.Do(int64(1))
			}
			wg.Done()
		}()
	}

	wg.Wait()
	if worker.Total != N*int64(setup.Procs) {
		t.Fatalf("got %v expected %v", worker.Total, N*setup.Procs)
	}
}

func testSumSequence(t *testing.T, setup *Setup) {
	const N = 100

	worker, combiner := setup.Make()
	defer StartClose(combiner)()

	var wg sync.WaitGroup

	wg.Add(setup.Procs)

	for proc := 0; proc < setup.Procs; proc++ {
		go func() {
			for i := int64(0); i < N; i++ {
				combiner.Do(i)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	if worker.Total != int64(setup.Procs)*N*(N-1)/2 {
		t.Fatalf("got %v expected %v", worker.Total, N*setup.Procs)
	}
}
