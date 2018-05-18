package combiner

type Batcher interface {
	Start()
	Do(arg interface{})
	Finish()
}
