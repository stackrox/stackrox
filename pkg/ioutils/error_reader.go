package ioutils

import (
	"io"
)

type errorReader struct {
	err error
}

// ErrorReader returns an io.Reader that returns the given error on any read attempt. If err is nil, it will return
// a reader returning io.EOF on any read.
func ErrorReader(err error) io.Reader {
	if err == nil {
		err = io.EOF
	}
	return errorReader{
		err: err,
	}
}

func (r errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}
