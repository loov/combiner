// +build ignore

package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/loov/combiner"
	"github.com/loov/hrtime"
)

const (
	P = 100
	N = 100
)

func main() {
	f, _ := os.Create("temp.dat~")
	f.Truncate(P * N)

	f.Seek(0, os.SEEK_SET)
	fmt.Println("CombiningFile", Bench(NewCombiningFile(f)))

	cfile := NewChanFile(f)
	cfile.Start()
	f.Seek(0, os.SEEK_SET)
	fmt.Println("ChanFile", Bench(cfile))
	cfile.Stop()

	f.Seek(0, os.SEEK_SET)
	fmt.Println("MutexFile", Bench(NewMutexFile(f)))

	// Output:
	// CombiningFile 163.661342ms
	// MutexFile 11.084999209s
}

type Writer interface {
	WriteByte(byte byte)
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

type CombiningFile struct {
	add  combiner.Parking
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

func (m *combiningAppend) Start()             { runtime.Gosched() }
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

type ChanFile struct {
	req  chan request
	file *os.File
}

type request struct {
	v    byte
	done chan struct{}
}

func NewChanFile(f *os.File) *ChanFile {
	m := &ChanFile{}
	m.req = make(chan request, 100)
	m.file = f
	return m
}

func (m *ChanFile) WriteByte(b byte) {
	r := request{b, make(chan struct{}, 0)}
	m.req <- r
	<-r.done
}

func (m *ChanFile) Start() {
	go func() {
		var requests = []request{}
		for {
			requests = requests[:0]
		combining:
			for count := 0; count < 100; count++ {
				select {
				case r := <-m.req:
					requests = append(requests, r)
					m.file.Write([]byte{r.v})
				default:
					break combining
				}
			}

			m.file.Sync()
			for _, req := range requests {
				close(req.done)
			}
		}
	}()
}

func (m *ChanFile) Stop() { close(m.req) }
