package testsuite

func StartClose(c Combiner) func() {
	if r, ok := c.(Runner); ok {
		go r.Run()
		return r.Close
	}
	return func() {}
}
