// Copyright (c) 2020 StackRox Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

package grpcweb

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/pkg/errors"
	"golang.stackrox.io/grpc-http1/internal/ioutils"
)

type errExtraData int64

func (e errExtraData) Error() string {
	return fmt.Sprintf("at least %d extra bytes after trailers frame", int64(e))
}

var (
	// ErrNoDecompressor means that we don't know how to decompress a compressed trailer message.
	ErrNoDecompressor = errors.New("compressed message encountered, but no decompressor specified")
)

// Decompressor returns a decompressed ReadCloser for a given compressed ReadCloser.
type Decompressor func(io.ReadCloser) io.ReadCloser

type responseReader struct {
	io.ReadCloser
	decompressor Decompressor
	trailers     *http.Header

	// err is the error condition encountered, if any (sticky!)
	err error

	// Indicates how many bytes of the current gRPC web message remain to be read. If 0, we expect the start of the next
	// message header.
	currMessageRemaining int64
	// A partially read message header
	currPartialMsgHeader []byte

	// partialTrailerData stores data read from a trailer in a previous read call.
	partialTrailerData []byte

	// Keeps track of whether we have read any data at all, and whether we have read trailers. Relevant for determining
	// whether we can accept an EOF.
	hasReadData, hasReadTrailers bool
}

// NewResponseReader returns a response reader that on-the-fly transcodes a gRPC web response into normal gRPC framing.
// Once the reader has reached EOF, the given trailers (which must be non-nil) are populated.
func NewResponseReader(origResp io.ReadCloser, trailers *http.Header, decompressor Decompressor) io.ReadCloser {
	return &responseReader{
		ReadCloser:   origResp,
		trailers:     trailers,
		decompressor: decompressor,
	}
}

func (r *responseReader) adjustResult(n int, err error) (int, error) {
	if r.hasReadTrailers {
		if n > 0 && (err == nil || err == io.EOF) {
			err = errExtraData(n)
		}
		n = 0
	} else if r.hasReadData && err == io.EOF /* && !r.hasReadTrailers */ {
		if len(r.partialTrailerData) > 0 {
			// If there are pending trailers, these will be handled in the next call to Read, hence do not propagate
			// EOF at this point. This is relevant if the reader returns EOF *with* the last bytes read, as opposed to
			// return `0, EOF` in a subsequent call.
			err = nil
		} else {
			err = io.ErrUnexpectedEOF
		}
	}
	return n, err
}

func (r *responseReader) Read(buf []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	n, err := r.adjustResult(r.doRead(buf))
	if err != nil {
		r.err = err
	}
	return n, err
}

func (r *responseReader) doRead(buf []byte) (int, error) {
	if len(r.partialTrailerData) > 0 {
		if err := r.readFullTrailers(); err != nil {
			return 0, err
		}
		r.hasReadTrailers = true
		r.partialTrailerData = nil
	}

	n, err := r.ReadCloser.Read(buf)
	if n > 0 {
		r.hasReadData = true
	}

	if r.hasReadTrailers {
		// If we have already read trailers, directly pass through the result. adjustResult will take care of
		// translating any extra data into a "real" error condition.
		return n, err
	}

	buf = buf[:n]
	nPayload := r.consume(buf)
	extraDataBytes := n - nPayload
	if extraDataBytes > 0 {
		r.partialTrailerData = append(r.partialTrailerData, buf[nPayload:n]...)
	}

	// Special case: read buffer only contains trailers. In this case, simply repeat the read.
	if nPayload == 0 && len(r.partialTrailerData) > 0 {
		return r.doRead(buf)
	}

	return nPayload, err
}

// readFullTrailers reads the trailers, taking the stored partial trailer data into account.
func (r *responseReader) readFullTrailers() error {
	reader := io.MultiReader(bytes.NewReader(r.partialTrailerData), r.ReadCloser)
	var frameHeader [5]byte
	_, err := io.ReadFull(reader, frameHeader[:])
	if err != nil {
		return err
	}

	frameLen := binary.BigEndian.Uint32(frameHeader[1:])
	var numBytesRead int64
	trailersDataReader := ioutils.NewCountingReader(io.LimitReader(reader, int64(frameLen)), &numBytesRead)
	if frameHeader[0]&compressedFlag != 0 {
		if r.decompressor == nil {
			return ErrNoDecompressor
		}
		trailersDataReader = r.decompressor(trailersDataReader)
	}

	// textproto Reader requires a terminating newline (\r\n) after the last header line, which is not contained in the
	// gRPC web trailer frame.
	trailersReader := textproto.NewReader(bufio.NewReader(
		io.MultiReader(trailersDataReader, strings.NewReader("\r\n"))))
	trailers, err := trailersReader.ReadMIMEHeader()
	if err != nil {
		return err
	}

	if _, err := trailersReader.R.Peek(1); err != io.EOF {
		if err == nil {
			err = errors.New("incomplete read of trailers")
		}
		return err
	}

	// Note that if we don't use a decompressor, this is guaranteed to not close the underlying reader, as `LimitReader`
	// will make the Close method inaccessible, and hence the reader returned by NewCountingReader doubles as a
	// NopCloser.
	if err := trailersDataReader.Close(); err != nil {
		return err
	}

	if numBytesRead != int64(frameLen) {
		return errors.Errorf("only read %d out of %d bytes from trailers frame", numBytesRead, frameLen)
	}

	r.populateTrailers(trailers)

	// Special case: if `r.partialTrailerData` contains data past the trailers frame, make sure we don't silently
	// discard it (we still discard it, but with an error).
	if extraBytes := int64(len(r.partialTrailerData)) - int64(len(frameHeader)) - int64(frameLen); extraBytes > 0 {
		return errExtraData(extraBytes)
	}

	return nil
}

func (r *responseReader) populateTrailers(trailers textproto.MIMEHeader) {
	if *r.trailers == nil {
		*r.trailers = make(http.Header)
	}

	for k, vs := range trailers {
		canonicalK := http.CanonicalHeaderKey(k)
		(*r.trailers)[canonicalK] = append((*r.trailers)[canonicalK], vs...)
	}
}

// consume reads regular frame data from buf, stopping as soon as the first byte of a trailer frame is encountered.
// The return value is the number of bytes consumed without any trailer frame data.
func (r *responseReader) consume(buf []byte) int {
	n := int64(0)
	for len(buf) > 0 {
		lastMsgBytes := r.currMessageRemaining
		if lastMsgBytes > int64(len(buf)) {
			lastMsgBytes = int64(len(buf))
		}
		buf = buf[lastMsgBytes:]
		r.currMessageRemaining -= lastMsgBytes
		n += lastMsgBytes

		if len(buf) == 0 {
			break
		}

		// At beginning of header - check if the next message is a trailer message
		if len(r.currPartialMsgHeader) == 0 {
			if buf[0]&trailerMessageFlag != 0 {
				break
			}
		}

		// Read header data
		remainingHeaderBytes := completeHeaderLen - len(r.currPartialMsgHeader)
		if remainingHeaderBytes > len(buf) {
			remainingHeaderBytes = len(buf)
		}

		r.currPartialMsgHeader = append(r.currPartialMsgHeader, buf[:remainingHeaderBytes]...)
		n += int64(remainingHeaderBytes)
		buf = buf[remainingHeaderBytes:]

		// Check for complete header
		if len(r.currPartialMsgHeader) == completeHeaderLen {
			r.currMessageRemaining = int64(binary.BigEndian.Uint32(r.currPartialMsgHeader[1:]))
			r.currPartialMsgHeader = r.currPartialMsgHeader[:0]
		}
	}

	return int(n)
}
