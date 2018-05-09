package extcombiner

import "github.com/loov/combiner/testsuite"

// All contains all combiner queue descriptions
var All = testsuite.Descs{
	{"Mutex", false, func(bat testsuite.Batcher, bound int) testsuite.Combiner { return NewMutex(bat) }},
	{"SpinMutex", false, func(bat testsuite.Batcher, bound int) testsuite.Combiner { return NewSpinMutex(bat) }},
	{"Basic", false, func(bat testsuite.Batcher, bound int) testsuite.Combiner { return NewBasic(bat) }},
	{"BasicS", false, func(bat testsuite.Batcher, bound int) testsuite.Combiner { return NewBasicSleepy(bat) }},
	{"BasicU", false, func(bat testsuite.Batcher, bound int) testsuite.Combiner { return NewBasicUintptr(bat) }},
	{"BasicSU", false, func(bat testsuite.Batcher, bound int) testsuite.Combiner { return NewBasicSleepyUintptr(bat) }},
	{"Bounded", true, func(bat testsuite.Batcher, bound int) testsuite.Combiner { return NewBounded(bat, bound) }},
	{"BoundedS", true, func(bat testsuite.Batcher, bound int) testsuite.Combiner { return NewBoundedSleepy(bat, bound) }},
	{"BoundedU", true, func(bat testsuite.Batcher, bound int) testsuite.Combiner { return NewBoundedUintptr(bat, bound) }},
	{"BoundedSU", true, func(bat testsuite.Batcher, bound int) testsuite.Combiner { return NewBoundedSleepyUintptr(bat, bound) }},
}
