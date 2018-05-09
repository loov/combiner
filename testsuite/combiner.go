package testsuite

type Combiner interface {
	Do(op interface{})
}

type Runner interface {
	Combiner
	Run()
	Close()
}

type AsyncCombiner interface {
	Combiner
	DoAsync(op interface{})
}

type Batcher interface {
	Start()
	Include(op interface{})
	Finish()
}

// Other possible designs
//   1. Include(v interface)
//   2. type Op func()
//   2. type Op interface { Execute() }
//   3. specialized
