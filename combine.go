package combiner

// Batcher is the operation combining implementation.
type Batcher interface {
	// Start is called on a start of a new batch.
	Start()
	// Do is called for each batch element.
	Do(arg interface{})
	// Finish is called after completing a batch.
	Finish()
}
