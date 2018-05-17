package combiner

import (
	"testing"

	"github.com/loov/combiner/testsuite"
)

// All contains all combiner queue descriptions
var All = testsuite.Descs{
	{"Spinning", true, func(bat testsuite.Batcher, bound int) testsuite.Combiner { return NewSpinning(bat, bound) }},
	{"Parking", true, func(bat testsuite.Batcher, bound int) testsuite.Combiner { return NewParking(bat, bound) }},
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
