package binenc

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
)

// WriteBytesList takes a list of byte slices and writes them in encoded form to a writer.
func WriteBytesList(w io.Writer, byteSlices ...[]byte) (int, error) {
	var total int
	for _, byteSlice := range byteSlices {
		n, err := WriteUVarInt(w, uint64(len(byteSlice)))
		total += n
		if err != nil {
			return total, err
		}

		n, err = w.Write(byteSlice)
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// EncodeBytesList takes a list of byte slices and encodes them into a single byte slice.
func EncodeBytesList(byteSlices ...[]byte) []byte {
	var buf bytes.Buffer
	_, _ = WriteBytesList(&buf, byteSlices...)
	return buf.Bytes()
}

// DecodeBytesList takes a byte buffer encoded via EncodeBytesList or WriteBytesList and decodes it into a list of byte
// slices.
func DecodeBytesList(buf []byte) ([][]byte, error) {
	var result [][]byte

	for len(buf) > 0 {
		sliceLen, l := binary.Uvarint(buf)
		if l <= 0 {
			return nil, errors.New("invalid varint in buffer")
		}
		buf = buf[l:]
		if sliceLen > uint64(len(buf)) {
			return nil, errors.Errorf("encountered varint %d which is larger than the remaining buffer size (%d)", sliceLen, len(buf))
		}
		result = append(result, buf[:sliceLen])
		buf = buf[sliceLen:]
	}
	return result, nil
}
