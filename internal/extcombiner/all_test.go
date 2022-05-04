package extcombiner

import (
	"testing"

	"loov.dev/combiner/internal/testsuite"
)

func Test(t *testing.T) {
	testsuite.Test.Iterate(All, func(setup *testsuite.Setup) {
		testsuite.RunTests(t, setup)
	})
}

func Benchmark(b *testing.B) {
	testsuite.Bench.Iterate(All, func(setup *testsuite.Setup) {
		testsuite.RunBenchmarks(b, setup)
	})
}
