package binenc

import (
	"encoding/binary"
	"io"
)

// WriteUVarInt writes the given unsigned integer in its varint representation to the specified writer. The return value
// is the result of calling `Write` with the corresponding byte buffer.
func WriteUVarInt(w io.Writer, x uint64) (int, error) {
	var buf [binary.MaxVarintLen64]byte
	l := binary.PutUvarint(buf[:], x)
	return w.Write(buf[:l])
}
