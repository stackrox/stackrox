package utils

// Must panics if the given error is non-nil, and does nothing otherwise.
func Must(err error) {
	if err != nil {
		panic(err)
	}
}
