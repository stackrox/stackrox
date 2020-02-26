package binenc

import (
	"encoding/binary"
	"io"
)

// VarInt converts the given integer to a varint representation and returns the result of a newly allocated byte slice
// of the exact size.
func VarInt(x int64) []byte {
	var buf [binary.MaxVarintLen64]byte
	l := binary.PutVarint(buf[:], x)
	data := make([]byte, l)
	copy(data, buf[:l])
	return data
}

// UVarInt converts the given unsigned integer to a varint representation and returns the result of a newly allocated
// byte slice of the exact size.
func UVarInt(x uint64) []byte {
	var buf [binary.MaxVarintLen64]byte
	l := binary.PutUvarint(buf[:], x)
	data := make([]byte, l)
	copy(data, buf[:l])
	return data
}

// WriteVarInt writes the given integer in its varint representation to the specified writer. The return value is the
// result of calling `Write` with the corresponding byte buffer.
func WriteVarInt(w io.Writer, x int64) (int, error) {
	var buf [binary.MaxVarintLen64]byte
	l := binary.PutVarint(buf[:], x)
	return w.Write(buf[:l])
}

// WriteUVarInt writes the given unsigned integer in its varint representation to the specified writer. The return value
// is the result of calling `Write` with the corresponding byte buffer.
func WriteUVarInt(w io.Writer, x uint64) (int, error) {
	var buf [binary.MaxVarintLen64]byte
	l := binary.PutUvarint(buf[:], x)
	return w.Write(buf[:l])
}
