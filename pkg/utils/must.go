package utils

// Must panics if any of the given errors is non-nil, and does nothing otherwise.
func Must(errs ...error) {
	for _, err := range errs {
		if err != nil {
			panic(err)
		}
	}
}
