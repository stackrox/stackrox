package ioutils

import "io"

type chainReader struct {
	currReader io.Reader
	nextReader func() io.Reader

	opts ChainReaderOpts
}

// ChainReaderOpts controls the behavior of a chained reader.
type ChainReaderOpts struct {
	// PropagateCloseErrors causes errors occurring during `Close`ing of readers to be reported by `Read`.
	PropagateCloseErrors bool
	// CloseAll ensures that even if `Close` is called prematurely, it will be called on all readers in the chain.
	CloseAll bool
}

// ChainReaders returns a reader that seamlessly chains the readers returned by subsequent calls to `nextReaderFn`,
// returning EOF once `nextReaderFn` returns `nil` for the first time. Any error encountered during `Read` on one of the
// returned readers will be returned as-is; in particular, if this error is sticky, the chain reader will return this
// error again and again as well.
// Note that the callback function does not allow returning an error; instead, return an `ErrorReader` if an error
// condition occurs while obtaining the next reader.
// The behavior (especially concerning `Close`s of underlying readers) can be controlled via the options.
func ChainReaders(nextReaderFn func() io.Reader, opts ChainReaderOpts) io.ReadCloser {
	return &chainReader{
		nextReader: nextReaderFn,
		opts:       opts,
	}
}

// ChainReadersLazy creates a new reader that concatenates the readers obtained from executing the given functions one
// after the other. The assumption is that if a function is not called, the respective reader is not created and hence
// doesn't need to be closed.
func ChainReadersLazy(readerFuncs ...func() io.Reader) io.ReadCloser {
	i := 0
	return &chainReader{
		nextReader: func() io.Reader {
			if i >= len(readerFuncs) {
				return nil
			}
			r := readerFuncs[i]()
			i++
			return r
		},
		opts: ChainReaderOpts{
			CloseAll: false,
		},
	}
}

// ChainReadersEager returns a reader that concatenates the contents of all given, already instantiated readers. When
// the reader is closed, *all* readers will be attempted to be closed as well.
func ChainReadersEager(readers ...io.Reader) io.ReadCloser {
	i := 0
	return &chainReader{
		nextReader: func() io.Reader {
			if i >= len(readers) {
				return nil
			}
			r := readers[i]
			i++
			return r
		},
		opts: ChainReaderOpts{
			CloseAll: true,
		},
	}
}

func (r *chainReader) Read(buf []byte) (int, error) {
	if r.nextReader == nil {
		return 0, io.EOF
	}

	n := 0
	var err error

	for n == 0 && err == nil {
		if r.currReader == nil {
			r.currReader = r.nextReader()
			if r.currReader == nil {
				r.nextReader = nil
				return 0, io.EOF
			}
		}
		n, err = r.currReader.Read(buf)
		if err == io.EOF {
			err = nil
			if closeErr := Close(r.currReader); closeErr != nil && r.opts.PropagateCloseErrors {
				err = closeErr
			}
			r.currReader = nil
		}
	}

	return n, err
}

func (r *chainReader) Close() error {
	var err error
	for r.nextReader != nil {
		if r.currReader == nil {
			if !r.opts.CloseAll {
				r.nextReader = nil
				return nil
			}

			r.currReader = r.nextReader()
			if r.currReader == nil {
				r.nextReader = nil
				return nil
			}
		}

		closeErr := Close(r.currReader)
		if closeErr != nil && r.opts.PropagateCloseErrors {
			// only propagate the first close error we encountered.
			if err == nil {
				err = closeErr
			}
		}
		r.currReader = nil
	}

	return err
}
