// +build ignore

package main

import (
	"bufio"
	"compress/zlib"
	"encoding/gob"
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/loov/combiner/testsuite"
	"github.com/loov/plot"
)

func main() {
	outputfile, err := os.Open("latency.zgob")
	if err != nil {
		log.Fatal(err)
	}
	defer outputfile.Close()

	bufferedfile := bufio.NewReader(outputfile)

	decompressor, _ := zlib.NewReader(bufferedfile)
	defer decompressor.Close()

	dec := gob.NewDecoder(decompressor)

	type Result struct {
		testsuite.Setup
		Results [][]time.Duration
	}
	results := make([]Result, 0, 1000)

	for {
		var r Result
		if err := dec.Decode(&r.Setup); err != nil {
			log.Println(err)
			break
		}
		if err := dec.Decode(&r.Results); err != nil {
			log.Println(err)
			break
		}

		results = append(results, r)
	}

	p := plot.New()
	plotStack := plot.NewVStack()
	plotStack.Margin = plot.R(0, 5, 0, 5)
	p.Add(plotStack)

	rowCount := 0.0
	colCount := 4.0

	axisGroups := map[int]*plot.AxisGroup{}

	var procStack *plot.HStack
	var axisGlobal *plot.AxisGroup
	var procGroup *plot.AxisGroup

	var previous Result

	p.X.Transform = plot.NewPercentileTransform(5)
	p.X.Ticks = plot.ManualTicks{
		{Value: 0, Label: "0"},
		{Value: 0.25, Label: "25"},
		{Value: 0.5, Label: "50"},
		{Value: 0.75, Label: "75"},
		{Value: 0.9, Label: "90"},
		{Value: 0.99, Label: "99"},
		{Value: 0.999, Label: "99.9"},
		{Value: 0.9999, Label: "99.99"},
	}

	sort.SliceStable(results, func(i, k int) bool {
		return results[i].Bounds < results[k].Bounds
	})

	for _, result := range results {
		if procStack == nil || result.Name != previous.Name || result.Bounds != previous.Bounds {
			nameStack := plot.NewHFlex()
			plotStack.Add(nameStack)

			procStack = plot.NewHStack()
			nameStack.Add(100, plot.NewTextbox(result.Name+":"+strconv.Itoa(result.Bounds)))
			nameStack.Add(0, procStack)

			procStack.Margin = plot.R(5, 0, 5, 0)
			rowCount++
		}

		if procGroup == nil || result.Procs != previous.Procs {
			if procGroup != nil {
				procGroup.Add(plot.NewTickLabels())
				procGroup.Add(plot.NewXLabel("P" + strconv.Itoa(previous.Procs)))
				procGroup.Update()
			}

			var ok bool
			axisGlobal, ok = axisGroups[result.Procs]
			if !ok {
				axisGlobal = plot.NewAxisGroup()
				axisGlobal.X = p.X
				axisGroups[result.Procs] = axisGlobal
			}

			procGroup = plot.NewAxisGroup()
			procGroup.X = p.X
			// procGroup.Y = axisGlobal.Y

			procStack.Add(procGroup)

			procGroup.Add(plot.NewGrid())
		}
		if result.WorkStart == 100 && result.WorkInclude == 0 && result.WorkFinish == 100 {
			all := []float64{}

			total := 0.0
			for _, r := range result.Results {
				rf := plot.DurationToNanoseconds(r)
				for i := range rf {
					total += rf[i]
				}
				all = append(all, rf...)
			}
			sort.Float64s(all)
			p := int(0.999 * float64(len(all)))
			fmt.Println(result.FullName(""), all[p])

			line := plot.NewPercentiles("", all)
			line.Stroke = color.NRGBA{0, 0, 0, 255}
			procGroup.Add(line)
			axisGlobal.Add(line)
		}
		previous = result
	}

	if procGroup != nil {
		procGroup.Add(plot.NewTickLabels())
		procGroup.Add(plot.NewXLabel("P" + strconv.Itoa(previous.Procs)))
		procGroup.Update()
	}

	for _, axisGroup := range axisGroups {
		axisGroup.Update()
	}

	svg := plot.NewSVG(150+400*colCount, 150*rowCount)
	p.Draw(svg)
	ioutil.WriteFile("latency.svg", svg.Bytes(), 0755)
}
