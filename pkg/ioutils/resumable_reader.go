package ioutils

import (
	"bytes"
	"hash"
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
)

// NewResumableReader returns a reader that may read from various underlying readers, one at a time. A reader is
// attached (either initially or until a detachment event), and detached once a call to Read on it returns an error.
// Detachment events are sent to a channel and MUST be processed, by either re-attaching a new reader or indicating that
// no further reader will be attached.
// The given checksumAlgo can be used to force checksum validations on resumes, to ensure that the stream of data is
// coming from the same source. If `nil` is passed, no checksum validations will be performed.
func NewResumableReader(checksumAlgo hash.Hash) (io.ReadCloser, ReaderAttachable, <-chan ReaderDetachmentEvent) {
	if checksumAlgo == nil {
		checksumAlgo = nilHash{}
	}
	checksumAlgo.Reset()

	cmdC := make(chan resumableReaderCmd, 1)
	eventsC := make(chan ReaderDetachmentEvent)

	r := &resumableReader{
		cmdC:              cmdC,
		eventsC:           eventsC,
		currChecksumState: checksumAlgo,
	}

	initAttachable := &readerAttachable{
		cmdC: cmdC,
	}

	return r, initAttachable, eventsC
}

// ReaderAttachable provides a mechanism to attach a reader to a resumable reader. Generally, every instance providing
// this interface must be used exactly once in an error-free way (i.e., after the invocation of any of the below methods
// with a nil return value, the object should be discarded).
type ReaderAttachable interface {
	// Finish indicates that no more readers will be attached. The given error (or io.EOF if err is nil) is presented to
	// the next caller of Read.
	Finish(err error) error
	// Attach attaches a new reader to the corresponding resumable reader. Attachment will only succeed if the reader is
	// attached at the right position, with the right partial checksum. For the initial reader attachment, the position
	// must be 0, and a nil checksum should be provided.
	Attach(newReader io.Reader, pos int64, checksum []byte) error
}

// ReaderDetachmentEvent is the event sent to a resumable reader controller. It contains the old reader as well as the
// error returned by a call to `Read` on the respective reader, and the last successfully read position. It furthermore
// allows re-attaching another reader, or terminating the stream.
type ReaderDetachmentEvent interface {
	DetachedReader() io.Reader
	ReadError() error

	Position() int64

	ReaderAttachable
}

type readerAttachable struct {
	used concurrency.Flag

	pos      int64
	checksum []byte

	cmdC chan<- resumableReaderCmd
}

func (a *readerAttachable) checkUsed() error {
	if a.used.TestAndSet(true) {
		return errors.New("Finish or Attach have already been called")
	}
	return nil
}

func (a *readerAttachable) Finish(finishErr error) error {
	if err := a.checkUsed(); err != nil {
		return err
	}

	if finishErr == nil {
		finishErr = io.EOF
	}
	a.cmdC <- resumableReaderCmd{finishErr: finishErr}
	close(a.cmdC)
	return nil
}

func (a *readerAttachable) Attach(r io.Reader, pos int64, checksum []byte) error {
	if a.pos != pos {
		return errors.Errorf("position mismatch when trying to attach reader: %d != %d", a.pos, pos)
	}
	if a.checksum != nil && !bytes.Equal(a.checksum, checksum) {
		return errors.New("checksum validation failed when trying to attach reader")
	}

	if err := a.checkUsed(); err != nil {
		return err
	}

	a.cmdC <- resumableReaderCmd{attachReader: r}
	return nil
}

type readerDetachmentEvent struct {
	readerAttachable

	detachedReader io.Reader
	readErr        error
}

func (e *readerDetachmentEvent) DetachedReader() io.Reader {
	return e.detachedReader
}

func (e *readerDetachmentEvent) ReadError() error {
	return e.readErr
}

func (e *readerDetachmentEvent) Position() int64 {
	return e.pos
}

type resumableReaderCmd struct {
	attachReader io.Reader
	finishErr    error
}

type resumableReader struct {
	closed concurrency.Flag

	currentReader io.Reader
	finishErr     error

	pos int64

	currChecksumState hash.Hash

	cmdC    chan resumableReaderCmd
	eventsC chan<- ReaderDetachmentEvent
}

func (r *resumableReader) Read(buf []byte) (int, error) {
	if len(buf) == 0 || r.finishErr != nil {
		return 0, r.finishErr
	}

	for r.finishErr == nil {
		if r.currentReader != nil {
			n, err := r.currentReader.Read(buf)
			r.processData(buf[:n])
			if err != nil {
				r.eventsC <- &readerDetachmentEvent{
					readerAttachable: readerAttachable{
						pos:      r.pos,
						checksum: r.currChecksumState.Sum(nil),
						cmdC:     r.cmdC,
					},
					detachedReader: r.currentReader,
					readErr:        err,
				}
				r.currentReader = nil
			}

			if n > 0 {
				// We could unconditionally return even, since `0, nil` would be interpreted as
				// "nothing happened, try again", but according to the doc on `Read`, this is discouraged.
				return n, nil
			}
		}

		// Wait for attachment or finish
		cmd := <-r.cmdC
		finishErr := cmd.finishErr
		if cmd.attachReader == nil && finishErr == nil {
			// The above also triggers if the channel was closed.
			finishErr = io.EOF
		}
		if finishErr == nil && cmd.attachReader != nil {
			r.currentReader = cmd.attachReader
		}
		r.finishErr = finishErr
	}

	if r.eventsC != nil {
		close(r.eventsC)
		r.eventsC = nil
	}
	return 0, r.finishErr
}

func (r *resumableReader) processData(buf []byte) {
	r.pos += int64(len(buf))
	_, _ = r.currChecksumState.Write(buf)
}

func (r *resumableReader) Close() error {
	if r.closed.TestAndSet(true) {
		return errors.New("already closed")
	}

	var err error
	if r.currentReader != nil {
		err = Close(r.currentReader)
		r.currentReader = nil
	}

	if r.eventsC != nil {
		close(r.eventsC)
		r.eventsC = nil
	}
	r.finishErr = errors.New("reader closed")

	return err
}
