package utils

// IgnoreError is useful when you want to defer a func that returns an error,
// but ignore the error.
func IgnoreError(f func() error) {
	if f != nil {
		_ = f()
	}
}
