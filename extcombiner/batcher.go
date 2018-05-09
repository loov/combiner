package extcombiner

type Batcher interface {
	Start()
	Include(op interface{})
	Finish()
}
