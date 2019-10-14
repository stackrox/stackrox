package ioutils

import "io"

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

// NopWriteCloser returns a WriteCloser that does nothing on Close.
func NopWriteCloser(w io.Writer) io.WriteCloser {
	return nopWriteCloser{
		Writer: w,
	}
}
