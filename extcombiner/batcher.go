package extcombiner

// Batcher combines different operations together and executes them.
type Batcher interface {
	Start()
	Do(op interface{})
	Finish()
}
