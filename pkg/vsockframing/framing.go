package vsockframing

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// ErrFrameTooLarge is returned by ReadFrame and DiscardFrame when the incoming
// frame's declared length exceeds the caller-supplied maxSize.
var ErrFrameTooLarge = errors.New("frame size exceeds limit")

// WriteFrame writes a length-prefixed frame: [4-byte big-endian uint32 length][payload].
func WriteFrame(w io.Writer, payload []byte) error {
	if uint64(len(payload)) > math.MaxUint32 {
		return fmt.Errorf("frame payload too large: %d bytes", len(payload))
	}
	length := uint32(len(payload))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return fmt.Errorf("writing frame length: %w", err)
	}
	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("writing frame payload: %w", err)
	}
	return nil
}

// ReadFrame reads a length-prefixed frame. Returns error if payload exceeds maxSize.
func ReadFrame(r io.Reader, maxSize uint32) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("reading frame length: %w", err)
	}
	if length > maxSize {
		return nil, fmt.Errorf("%w: %d exceeds limit %d", ErrFrameTooLarge, length, maxSize)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("reading frame payload: %w", err)
	}
	return payload, nil
}

// DiscardFrame drains one length-prefixed frame without allocating a payload
// buffer, for callers that only need to consume the frame (e.g. absorb on reject).
func DiscardFrame(r io.Reader, maxSize uint32) error {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return fmt.Errorf("reading frame length: %w", err)
	}
	if length > maxSize {
		return fmt.Errorf("%w: %d exceeds limit %d", ErrFrameTooLarge, length, maxSize)
	}
	n, err := io.CopyN(io.Discard, r, int64(length))
	if err != nil {
		// CopyN returns EOF on a short read; ReadFrame uses ReadFull, which
		// returns ErrUnexpectedEOF — match that so callers can treat both alike.
		if errors.Is(err, io.EOF) && n < int64(length) {
			err = io.ErrUnexpectedEOF
		}
		return fmt.Errorf("discarding frame payload: %w", err)
	}
	return nil
}
