// +build ignore

package main

import (
	"bufio"
	"compress/zlib"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/loov/combiner/extcombiner"
	"github.com/loov/combiner/testsuite"
	"github.com/loov/hrtime"
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
	const K = 10

	params := testsuite.Params{
		Procs:  []int{1, 4, 32, 256},
		Bounds: []int{4, 8, 16, 64},

		WorkStart:   []int{0, 100},
		WorkInclude: []int{0},
		WorkFinish:  []int{0, 100},
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
		for _, hr := range hrs {
			laps := hr.Laps()
			count += len(laps)
			for _, lap := range laps {
				average += lap
			}
			results = append(results, laps)
		}
		fmt.Println(average / time.Duration(count))
		enc.Encode(results)
	})
}
