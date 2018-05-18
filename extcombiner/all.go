package extcombiner

import "github.com/loov/combiner/testsuite"

// All contains all combiner queue descriptions
var All = testsuite.Descs{
	{"Mutex", false, func(bat testsuite.Batcher, bound int) testsuite.Combiner {
		return NewMutex(bat)
	}},
	// {"SpinMutex", false, func(bat testsuite.Batcher, bound int) testsuite.Combiner {
	// return NewSpinMutex(bat) }},
	{"BasicSpinning", false, func(bat testsuite.Batcher, bound int) testsuite.Combiner {
		return NewBasicSpinning(bat)
	}},
	{"BasicSpinningUintptr", false, func(bat testsuite.Batcher, bound int) testsuite.Combiner {
		return NewBasicSpinningUintptr(bat)
	}},
	{"BasicParking", false, func(bat testsuite.Batcher, bound int) testsuite.Combiner {
		return NewBasicParking(bat)
	}},
	{"BasicParkingUintptr", false, func(bat testsuite.Batcher, bound int) testsuite.Combiner {
		return NewBasicParkingUintptr(bat)
	}},
	{"BoundedSpinning", true, func(bat testsuite.Batcher, bound int) testsuite.Combiner {
		return NewBoundedSpinning(bat, bound)
	}},
	{"BoundedSpinningUintptr", true, func(bat testsuite.Batcher, bound int) testsuite.Combiner {

		return NewBoundedSpinningUintptr(bat, bound)
	}},
	{"BoundedParking", true, func(bat testsuite.Batcher, bound int) testsuite.Combiner {
		return NewBoundedParking(bat, bound)
	}},
	{"BoundedParkingUintptr", true, func(bat testsuite.Batcher, bound int) testsuite.Combiner {
		return NewBoundedParkingUintptr(bat, bound)
	}},
}
