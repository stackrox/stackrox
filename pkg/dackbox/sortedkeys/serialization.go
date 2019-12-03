package sortedkeys

import (
	"errors"

	"github.com/stackrox/rox/pkg/binenc"
)

// Unmarshal unmarshals a set of SortedKeys.
func Unmarshal(marshalled []byte) (SortedKeys, error) {
	var unmarshalled SortedKeys
	buf := marshalled
	for len(buf) >= 2 {
		// First two bytes encode the length.
		length := decodeLength(buf[:2])
		buf = buf[2:]
		if length > len(buf) {
			return nil, errors.New("malformed sorted keys, position out of range")
		}
		// Next length bytes encode the key.
		unmarshalled = append(unmarshalled, buf[:length])
		buf = buf[length:]
	}
	if len(buf) > 0 {
		return nil, errors.New("bytes remaining after unmarshal")
	}
	return unmarshalled, nil
}

// Marshal marshals the sorted keys.
func (sk SortedKeys) Marshal() []byte {
	var marshalled []byte
	for _, key := range sk {
		encodedLength := encodeLength(len(key))
		marshalled = append(marshalled, encodedLength...)
		marshalled = append(marshalled, key...)
	}
	return marshalled
}

func decodeLength(b []byte) int {
	return int(binenc.BigEndian.Uint16(b))
}

func encodeLength(length int) []byte {
	return binenc.BigEndian.EncodeUint16(uint16(length))
}
