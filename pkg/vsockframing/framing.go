package vsockframing

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

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
		return nil, fmt.Errorf("frame size %d exceeds limit %d", length, maxSize)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("reading frame payload: %w", err)
	}
	return payload, nil
}
