package testsuite

import (
	"encoding/gob"
	"fmt"
	"testing"
)

var (
	Test = Params{
		Procs:      []int{1, 2, 3, 4, 8, 16, 32, 64},
		Bounds:     []int{1, 2, 3, 4, 5, 6, 7, 8},
		WorkStart:  []int{0},
		WorkDo:     []int{0},
		WorkFinish: []int{0},
	}

	Bench = Params{
		Procs:      []int{1, 4, 32, 256},
		Bounds:     []int{4, 8, 16, 64},
		WorkStart:  []int{0},
		WorkDo:     []int{0},
		WorkFinish: []int{0, 100},
	}
)

type Desc struct {
	Name    string
	Bounded bool
	Create  func(exe Batcher, bound int) Combiner
}

type Descs []Desc

type Params struct {
	Procs      []int
	Bounds     []int
	WorkStart  []int
	WorkDo     []int
	WorkFinish []int
}

func (params *Params) Iterate(descs Descs, fn func(*Setup)) {
	setup := Setup{}
	for _, desc := range descs {
		setup.Name = desc.Name
		setup.Create = desc.Create

		bounds := params.Bounds
		if !desc.Bounded {
			bounds = []int{0}
		}

		for _, setup.Bounds = range bounds {
			for _, setup.Procs = range params.Procs {
				for _, setup.WorkStart = range params.WorkStart {
					for _, setup.WorkDo = range params.WorkDo {
						for _, setup.WorkFinish = range params.WorkFinish {
							tmp := setup
							fn(&tmp)
						}
					}
				}
			}
		}
	}
}

type Setup struct {
	Name       string
	Create     func(exe Batcher, bound int) Combiner
	Bounds     int
	Procs      int
	WorkStart  int
	WorkDo     int
	WorkFinish int
}

func init() { gob.Register(Setup{}) }

func (setup *Setup) Make() (*Worker, Combiner) {
	worker := &Worker{}
	worker.WorkStart = setup.WorkStart
	worker.WorkDo = setup.WorkDo
	worker.WorkFinish = setup.WorkFinish
	combiner := setup.Create(worker, setup.Bounds)
	return worker, combiner
}

func (setup *Setup) FullName(test string) string {
	return fmt.Sprintf("%v/b%v/%v/p%vs%vi%vr%v",
		setup.Name,
		setup.Bounds,
		test,
		setup.Procs,
		setup.WorkStart,
		setup.WorkDo,
		setup.WorkFinish,
	)
}

func (setup *Setup) Test(t *testing.T, name string, test func(t *testing.T, setup *Setup)) {
	t.Helper()
	t.Run(setup.FullName(name), func(t *testing.T) {
		test(t, setup)
	})
}

func (setup *Setup) Bench(b *testing.B, name string, bench func(b *testing.B, setup *Setup)) {
	b.Helper()
	b.Run(setup.FullName(name), func(b *testing.B) {
		bench(b, setup)
	})
}
