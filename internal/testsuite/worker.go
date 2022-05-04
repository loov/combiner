package testsuite

import (
	"runtime"
	"time"
)

type Worker struct {
	WorkStart  int
	WorkDo     int
	WorkFinish int

	SleepStart  time.Duration
	SleepDo     time.Duration
	SleepFinish time.Duration

	Total   int64
	Batches int64
}

func NewWorker() *Worker { return &Worker{} }

func (exe *Worker) Start() {
	simulateWork(exe.WorkStart, exe.SleepStart)
}

func (exe *Worker) Do(v interface{}) {
	exe.Total += v.(int64)
	simulateWork(exe.WorkDo, exe.SleepDo)
}

func (exe *Worker) Finish() {
	exe.Batches++
	simulateWork(exe.WorkFinish, exe.SleepFinish)
}

func simulateWork(amount int, sleep time.Duration) {
	foo := 1
	for i := 0; i < amount; i++ {
		foo *= 2
		foo /= 2
	}
	if amount > 0 {
		runtime.Gosched()
	}
	if sleep > 0 {
		time.Sleep(sleep)
	}
}
