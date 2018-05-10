package extcombiner

import (
	"bufio"
	"encoding/gob"
	"flag"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/loov/combiner/testsuite"
	"github.com/loov/hrtime"
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

var latency = flag.String("latency", "", "latency measurement output")

func TestLatency(t *testing.T) {
	if *latency == "" {
		t.Skip("latency measurements disabled")
	}

	outputfile, err := os.Create(*latency)
	if err != nil {
		t.Fatal(err)
	}
	defer outputfile.Close()
	output := bufio.NewWriter(outputfile)
	defer output.Flush()

	enc := gob.NewEncoder(output)

	const N = 1000
	const K = 10

	params := testsuite.Params{
		Procs:  []int{1, 4, 32, 256},
		Bounds: []int{4, 8, 16, 64},

		WorkStart:   []int{0},
		WorkInclude: []int{0},
		WorkFinish:  []int{0, 100},
	}

	params.Iterate(All, func(setup *testsuite.Setup) {
		t.Run(setup.FullName("Basic"), func(t *testing.T) {
			hrs := make([]*hrtime.BenchmarkTSC, setup.Procs)
			for i := range hrs {
				hrs[i] = hrtime.NewBenchmarkTSC(N)
			}

			var wg sync.WaitGroup
			wg.Add(setup.Procs)

			_, combiner := setup.Make()
			defer testsuite.StartClose(combiner)()

			for i := 0; i < setup.Procs; i++ {
				go func(b *hrtime.BenchmarkTSC) {
					defer wg.Done()
					v := int64(0)
					for b.Next() {
						for k := 0; k < K; k++ {
							combiner.Do(v)
							v++
						}
					}
				}(hrs[i])
			}

			wg.Wait()

			enc.Encode(setup)

			var results [][]time.Duration
			for _, hr := range hrs {
				results = append(results, hr.Laps())
			}
			enc.Encode(results)
		})
	})
}
