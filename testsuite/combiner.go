package testsuite

type Combiner interface {
	Do(op Argument)
}

type Runner interface {
	Combiner
	Run()
	Close()
}

type AsyncCombiner interface {
	Combiner
	DoAsync(op Argument)
}

type Batcher interface {
	Start()
	Include(op Argument)
	Finish()
}

type Argument interface{}

// Other possible designs
//   1. Include(v interface)
//   2. type Op func()
//   2. type Op interface { Execute() }
//   3. specialized
