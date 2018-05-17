package combiner

type Batcher interface {
	Start()
	Include(argument interface{})
	Finish()
}
