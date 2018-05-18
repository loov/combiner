package combiner

type Batcher interface {
	Start()
	Do(arg interface{})
	Finish()
}

type IncludeFunc func(arg interface{})

func (fn IncludeFunc) Start()             {}
func (fn IncludeFunc) Do(arg interface{}) { fn(arg) }
func (fn IncludeFunc) Finish()            {}
