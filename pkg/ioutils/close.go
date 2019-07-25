package ioutils

import "io"

// Close closes the given object if it implements the `io.Closer` interface, and returns any error that might occur
// in that process. If the object does not implement the `io.Closer` interface, `nil` is returned.
func Close(x interface{}) error {
	if c, _ := x.(io.Closer); c != nil {
		return c.Close()
	}
	return nil
}
