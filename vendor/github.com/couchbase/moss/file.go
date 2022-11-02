//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the
//  License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing,
//  software distributed under the License is distributed on an "AS
//  IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
//  express or implied. See the License for the specific language
//  governing permissions and limitations under the License.

package moss

import (
	"io"
	"os"
	"sync"
)

// An InitCloser holds onto an io.Closer, and is used for chaining
// io.Closer's.  That is, we often want the closing of one resource to
// close related resources.
type InitCloser interface {
	InitCloser(io.Closer) error
}

// The File interface is implemented by os.File.  App specific
// implementations may add concurrency, caching, stats, fuzzing, etc.
type File interface {
	io.ReaderAt
	io.WriterAt
	io.Closer
	Stat() (os.FileInfo, error)
	Sync() error
	Truncate(size int64) error
}

// The OpenFile func signature is similar to os.OpenFile().
type OpenFile func(name string, flag int, perm os.FileMode) (File, error)

// FileRef provides a ref-counting wrapper around a File.
type FileRef struct {
	file File
	m    sync.Mutex // Protects the fields that follow.
	refs int

	beforeCloseCallbacks []func() // Optional callbacks invoked before final close.
	afterCloseCallbacks  []func() // Optional callbacks invoked after final close.
}

type ioResult struct {
	kind string // Kind of io attempted.
	want int    // Num bytes expected to be written or read.
	got  int    // Num bytes actually written or read.
	err  error
}

// --------------------------------------------------------

// OnBeforeClose registers event callback func's that are invoked before the
// file is closed.
func (r *FileRef) OnBeforeClose(cb func()) {
	r.m.Lock()
	r.beforeCloseCallbacks = append(r.beforeCloseCallbacks, cb)
	r.m.Unlock()
}

// OnAfterClose registers event callback func's that are invoked after the
// file is closed.
func (r *FileRef) OnAfterClose(cb func()) {
	r.m.Lock()
	r.afterCloseCallbacks = append(r.afterCloseCallbacks, cb)
	r.m.Unlock()
}

// AddRef increases the ref-count on the file ref.
func (r *FileRef) AddRef() File {
	if r == nil {
		return nil
	}

	r.m.Lock()
	r.refs++
	file := r.file
	r.m.Unlock()

	return file
}

// DecRef decreases the ref-count on the file ref, and closing the
// underlying file when the ref-count reaches zero.
func (r *FileRef) DecRef() (err error) {
	if r == nil {
		return nil
	}

	r.m.Lock()

	r.refs--
	if r.refs <= 0 {
		for _, cb := range r.beforeCloseCallbacks {
			cb()
		}
		r.beforeCloseCallbacks = nil

		err = r.file.Close()

		for _, cb := range r.afterCloseCallbacks {
			cb()
		}
		r.afterCloseCallbacks = nil

		r.file = nil
	}

	r.m.Unlock()

	return err
}

// Close allows the FileRef to implement the io.Closer interface.  It actually
// just performs what should be the final DecRef() call which takes the
// reference count to 0.  Once 0, it allows the file to actually be closed.
func (r *FileRef) Close() error {
	return r.DecRef()
}

// FetchRefCount fetches the ref-count on the file ref.
func (r *FileRef) FetchRefCount() int {
	if r == nil {
		return 0
	}

	r.m.Lock()
	ref := r.refs
	r.m.Unlock()

	return ref
}

// --------------------------------------------------------

// OsFile interface allows conversion from a File to an os.File.
type OsFile interface {
	OsFile() *os.File
}

// ToOsFile provides the underlying os.File for a File, if available.
func ToOsFile(f File) *os.File {
	if osFile, ok := f.(*os.File); ok {
		return osFile
	}
	if osFile2, ok := f.(OsFile); ok {
		return osFile2.OsFile()
	}
	return nil
}

// --------------------------------------------------------

type bufferedSectionWriter struct {
	err error
	w   io.WriterAt
	beg int64 // Start position where we started writing in file.
	cur int64 // Current write-at position in file.
	max int64 // When > 0, max number of bytes we can write.
	buf []byte
	n   int

	stopCh chan struct{}
	doneCh chan struct{}
	reqCh  chan ioBuf
	resCh  chan ioBuf
}

type ioBuf struct {
	buf []byte
	pos int64
	err error
}

// newBufferedSectionWriter converts incoming Write() requests into
// buffered, asynchronous WriteAt()'s in a section of a file.
func newBufferedSectionWriter(w io.WriterAt, begPos, maxBytes int64,
	bufSize int, s statsReporter) *bufferedSectionWriter {
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	reqCh := make(chan ioBuf)
	resCh := make(chan ioBuf)

	go func() {
		defer close(doneCh)
		defer close(resCh)

		buf := make([]byte, bufSize)
		var pos int64
		var err error

		for {
			select {
			case <-stopCh:
				return
			case resCh <- ioBuf{buf: buf, pos: pos, err: err}:
			}

			req, ok := <-reqCh
			if ok {
				buf, pos = req.buf, req.pos
				if len(buf) > 0 {
					nBytes, err := w.WriteAt(buf, pos)
					if err == nil && s != nil {
						s.reportBytesWritten(uint64(nBytes))
					}
				}
			}
		}
	}()

	return &bufferedSectionWriter{
		w:   w,
		beg: begPos,
		cur: begPos,
		max: maxBytes,
		buf: make([]byte, bufSize),

		stopCh: stopCh,
		doneCh: doneCh,
		reqCh:  reqCh,
		resCh:  resCh,
	}
}

// Offset returns the byte offset into the file where the
// bufferedSectionWriter is currently logically positioned.
func (b *bufferedSectionWriter) Offset() int64 { return b.cur + int64(b.n) }

// Written returns the logical number of bytes written to this
// bufferedSectionWriter; or, the sum of bytes to Write() calls.
func (b *bufferedSectionWriter) Written() int64 { return b.Offset() - b.beg }

func (b *bufferedSectionWriter) Write(p []byte) (nn int, err error) {
	if b.max > 0 && b.Written()+int64(len(p)) > b.max {
		return 0, io.ErrShortBuffer // Would go over b.max.
	}
	for len(p) > 0 && b.err == nil {
		n := copy(b.buf[b.n:], p)
		b.n += n
		nn += n
		if n < len(p) {
			b.err = b.Flush()
		}
		p = p[n:]
	}
	return nn, b.err
}

func (b *bufferedSectionWriter) Flush() error {
	if b.err != nil {
		return b.err
	}
	if b.n <= 0 {
		return nil
	}

	prevWrite := <-b.resCh
	b.err = prevWrite.err
	if b.err != nil {
		return b.err
	}

	b.reqCh <- ioBuf{buf: b.buf[0:b.n], pos: b.cur}

	b.cur += int64(b.n)
	b.buf = prevWrite.buf[:]
	b.n = 0

	return nil
}

func (b *bufferedSectionWriter) Stop() error {
	if b.stopCh != nil {
		close(b.stopCh)
		close(b.reqCh)
		<-b.doneCh
		b.stopCh = nil
	}
	return b.err
}
