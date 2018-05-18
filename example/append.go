// +build ignore

package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/loov/combiner"
	"github.com/loov/hrtime"
)

const (
	P = 100
	N = 100
)

type CombiningFile struct {
	add  combiner.Spinning
	file *os.File
}
type combiningAppend CombiningFile

func NewCombiningFile(f *os.File) *CombiningFile {
	m := &CombiningFile{}
	m.add.Init((*combiningAppend)(m), 100)
	m.file = f
	return m
}

func (m *CombiningFile) WriteByte(b byte) { m.add.Do(b) }

func (m *combiningAppend) Start()             {}
func (m *combiningAppend) Do(arg interface{}) { m.file.Write([]byte{arg.(byte)}) }
func (m *combiningAppend) Finish()            { m.file.Sync() }

type MutexFile struct {
	mu   sync.Mutex
	file *os.File
}

func NewMutexFile(f *os.File) *MutexFile {
	m := &MutexFile{}
	m.file = f
	return m
}

func (m *MutexFile) WriteByte(b byte) {
	m.mu.Lock()
	m.file.Write([]byte{b})
	m.file.Sync()
	m.mu.Unlock()
}

type Writer interface {
	WriteByte(byte byte)
}

func main() {
	f, _ := os.Create("temp.dat~")
	f.Truncate(P * N)

	f.Seek(0, os.SEEK_SET)
	fmt.Println("CombiningFile", Bench(NewCombiningFile(f)))

	f.Seek(0, os.SEEK_SET)
	fmt.Println("MutexFile", Bench(NewMutexFile(f)))

	// Output:
	// CombiningFile 163.661342ms
	// MutexFile 11.084999209s
}

func Bench(w Writer) time.Duration {
	start := hrtime.TSC()

	var wg sync.WaitGroup
	wg.Add(P)
	for i := 0; i < P; i++ {
		go func(pid int) {
			for i := 0; i < N; i++ {
				w.WriteByte(byte(i))
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	stop := hrtime.TSC()
	return (stop - start).ApproxDuration()
}
