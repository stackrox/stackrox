package ioutils

import "io"

type readerWithCloser struct {
	io.Reader
	closer func() error
}

func (r *readerWithCloser) Close() error {
	return r.closer()
}

// ReaderWithCloser returns an `io.ReadCloser` that reads from the given reader, and invokes the given `closer`
// callback upon `Close()`.
// Note: if `r` is already an `io.ReadCloser`, invoking `Close()` on the returned object will NOT invoke `Close()` on
// the underlying `io.ReadCloser`.
func ReaderWithCloser(r io.Reader, closer func() error) io.ReadCloser {
	return &readerWithCloser{
		Reader: r,
		closer: closer,
	}
}
