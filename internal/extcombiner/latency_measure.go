//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"compress/zlib"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/loov/hrtime"
	"loov.dev/combiner/internal/extcombiner"
	"loov.dev/combiner/internal/testsuite"
)

func main() {
	outputfile, err := os.Create("latency2.zgob")
	if err != nil {
		log.Fatal(err)
	}
	defer outputfile.Close()

	bufferedfile := bufio.NewWriter(outputfile)
	defer bufferedfile.Flush()

	compressor := zlib.NewWriter(bufferedfile)
	defer compressor.Close()

	enc := gob.NewEncoder(compressor)

	const N = 1000
	const K = 1

	params := testsuite.Params{
		Procs:  []int{1, 4, 32, 256},
		Bounds: []int{64},

		WorkStart:  []int{100},
		WorkDo:     []int{100},
		WorkFinish: []int{1000},
	}

	params.Iterate(extcombiner.All, func(setup *testsuite.Setup) {
		fmt.Print(setup.FullName(""), "\t")
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

		average := time.Duration(0)
		count := 0
		var results [][]time.Duration
		var all []time.Duration
		for _, hr := range hrs {
			laps := hr.Laps()
			count += len(laps)
			all = append(all, laps...)
			for _, lap := range laps {
				average += lap
			}
			results = append(results, laps)
		}

		sort.Slice(all, func(i, k int) bool { return all[i] < all[k] })
		p := int(0.9999 * float64(len(all)))
		fmt.Println(average/time.Duration(count), "\t", all[p])
		enc.Encode(results)
	})
}
