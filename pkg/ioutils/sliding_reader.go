package ioutils

import (
	"encoding"
	"hash"
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/ringbuffer"
	"github.com/stackrox/stackrox/pkg/utils"
)

type marshallableHash interface {
	hash.Hash
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type slidingReaderWithChecksum struct {
	startOfWindowChecksumState marshallableHash
	tempChecksumState          marshallableHash

	createReader  func() io.Reader
	currReader    io.Reader
	currReaderPos int64

	ringBuf    *ringbuffer.RingBuffer
	readOffset int // offset between the position of the underlying reader and the logical read position.
}

// NewSlidingReader returns a reader that implements the `SeekableReaderWithChecksum` interface. It does so by
// maintaining a window (of a given size) of up to the last `maxWindowSize` bytes read. The checksum computation takes
// place from the beginning of this window, such that rewinding to a position inside the window does not require any
// action on the underlying reader. Positions before the current window however require the reader to be rewinded from
// the beginning, which is done either by invoking the `Seek` method if the reader implements the `ReadSeeker`
// interface, or closing the current reader and creating a new one via a call to the given callback.
// The `createChecksumAlgo` function must create a hash that is used for checksum computation; the returned hash must
// implement the Binary(Un)Marshaller interfaces. It is legal to pass `nil` as the `createChecksumAlgo` function,
// in which case all calls to CurrentChecksum will return nil as well.
// This reader is not safe to be accessed concurrently; if concurrently accessing it from multiple goroutines, all
// access has to be synchronized externally.
// CAVEAT: The Seek function only supports `io.SeekStart` and `io.SeekCurrent` as valid `whence` parameters;
// `io.SeekEnd` is not supported.
func NewSlidingReader(createReader func() io.Reader, maxWindowSize int, createChecksumAlgo func() hash.Hash) (SeekableReaderWithChecksum, error) {
	var mh, mhCopy marshallableHash
	if createChecksumAlgo != nil {
		mh, _ = createChecksumAlgo().(marshallableHash)
		mhCopy, _ = createChecksumAlgo().(marshallableHash)
	} else {
		mh, mhCopy = nilHash{}, nilHash{}
	}
	if mh == nil || mhCopy == nil {
		return nil, errors.New("checksum algorithm does not implement the Binary(Un)Marshaler interfaces")
	}

	mh.Reset()

	return &slidingReaderWithChecksum{
		startOfWindowChecksumState: mh,
		tempChecksumState:          mhCopy,
		createReader:               createReader,
		currReader:                 createReader(),
		currReaderPos:              0,
		ringBuf:                    ringbuffer.NewRingBuffer(maxWindowSize),
	}, nil
}

func (r *slidingReaderWithChecksum) resetReader() error {
	if seeker, _ := r.currReader.(io.ReadSeeker); seeker != nil {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return err
		}
	} else {
		if err := Close(r.currReader); err != nil {
			return errors.Wrap(err, "failed to close current reader")
		}
		r.currReader = r.createReader()
	}

	r.ringBuf.Reset(nil)
	r.currReaderPos = 0
	r.startOfWindowChecksumState.Reset()
	r.readOffset = 0
	return nil
}

func (r *slidingReaderWithChecksum) checksumStateCopy() hash.Hash {
	data, err := r.startOfWindowChecksumState.MarshalBinary()
	utils.CrashOnError(err) // should not happen for hashes
	utils.Must(r.tempChecksumState.UnmarshalBinary(data))
	return r.tempChecksumState
}

func (r *slidingReaderWithChecksum) CurrentChecksum() []byte {
	numBytesFromBuffer := r.ringBuf.Size() - r.readOffset
	if numBytesFromBuffer < 0 {
		panic(errors.Errorf("UNEXPECTED: readOffset (%d) is larger than buffer (%d)", r.readOffset, r.ringBuf.Size()))
	}
	if numBytesFromBuffer == 0 {
		return r.startOfWindowChecksumState.Sum(nil)
	}

	cs := r.checksumStateCopy()
	for _, chunk := range r.ringBuf.ReadFirst(numBytesFromBuffer) {
		_, _ = cs.Write(chunk)
	}
	return cs.Sum(nil)
}

func (r *slidingReaderWithChecksum) Seek(offset int64, whence int) (int64, error) {
	var newPos int64

	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = r.currReaderPos - int64(r.readOffset) + offset
	default:
		return 0, errors.Errorf("unsupported whence value %d", whence)
	}

	if newPos < 0 {
		return 0, errors.New("seeking past the beginning of the stream is illegal")
	}

	// Good news! Position is inside the window covered by our buffer.
	if offset := r.currReaderPos - newPos; offset >= 0 && offset <= int64(r.ringBuf.Size()) {
		r.readOffset = int(offset)
		return newPos, nil
	}

	if newPos < r.currReaderPos {
		if err := r.resetReader(); err != nil {
			return 0, err
		}
	}

	// Assume:
	//   newPos >= r.currReaderPos

	forwardBy := newPos - r.currReaderPos
	r.readOffset = 0

	// Determine the amount of data that for sure won't fit into the ring buffer, and just use it for checksum
	// computation.
	if toSkip := forwardBy - int64(r.ringBuf.Capacity()); toSkip > 0 {
		r.ringBuf.Reset(r.updateStartOfWindowChecksumState)

		n, err := io.CopyN(r.startOfWindowChecksumState, r.currReader, toSkip)
		r.currReaderPos += n
		if err != nil {
			if err == io.EOF {
				// Not an error, but we are now at the end of the stream and have reduced our window to zero...
				return r.currReaderPos, nil
			}
			return r.currReaderPos, err
		}
	}

	// Assume:
	//   newPos - r.currReaderPos <= r.buf.Capacity()
	//   end of ring buffer is aligned with r.currReaderPos
	//   r.readOffset == 0

	lastChunk := make([]byte, newPos-r.currReaderPos)
	n, err := io.ReadFull(r.currReader, lastChunk)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		err = nil
	}
	r.currReaderPos += int64(n)
	r.ringBuf.Write(lastChunk[:n], r.updateStartOfWindowChecksumState)

	return r.currReaderPos, err
}

func (r *slidingReaderWithChecksum) Read(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	if r.readOffset > 0 {
		n := r.readOffset
		if n > len(buf) {
			n = len(buf)
		}

		for _, chunk := range r.ringBuf.Read(-r.readOffset, n) {
			copy(buf[:len(chunk)], chunk)
			buf = buf[len(chunk):]
		}

		r.readOffset -= n
		return n, nil
	}

	n, err := r.currReader.Read(buf)
	if n > 0 {
		r.ringBuf.Write(buf[:n], r.updateStartOfWindowChecksumState)
		r.currReaderPos += int64(n)
	}
	return n, err
}

func (r *slidingReaderWithChecksum) Close() error {
	if r.createReader == nil {
		return errors.New("already closed")
	}

	r.createReader = nil
	if r.currReader == nil {
		return nil
	}
	return Close(r.currReader)
}

func (r *slidingReaderWithChecksum) updateStartOfWindowChecksumState(chunk []byte) {
	_, _ = r.startOfWindowChecksumState.Write(chunk)
}
