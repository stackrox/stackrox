package ioutils

import (
	"io"
)

// A SeekableReaderWithChecksum is a reader that can be moved to any valid position, while providing access to a
// checksum of the partial contents up to that position. This obviously implies that moving to a position cannot happen
// in a pure random-access way.
type SeekableReaderWithChecksum interface {
	io.ReadSeeker
	io.Closer

	// CurrentChecksum returns the checksum at the current position.
	CurrentChecksum() []byte
}
