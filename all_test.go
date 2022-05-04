package combiner

import (
	"testing"

	"loov.dev/combiner/internal/testsuite"
)

// All contains all combiner queue descriptions
var All = testsuite.Descs{
	{
		Name:    "Parking",
		Bounded: true,
		Create: func(bat testsuite.Batcher, bound int) testsuite.Combiner {
			return New[interface{}](bat, bound)
		},
	},
}

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
