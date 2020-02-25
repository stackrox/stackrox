package sortedkeys

import (
	"encoding/binary"
	"errors"
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
	encodedLength := make([]byte, 2)
	for _, key := range sk {
		encodeLength(len(key), encodedLength)
		marshalled = append(marshalled, encodedLength...)
		marshalled = append(marshalled, key...)
	}
	return marshalled
}

func decodeLength(b []byte) int {
	return int(binary.BigEndian.Uint16(b))
}

func encodeLength(length int, encodedLength []byte) {
	binary.BigEndian.PutUint16(encodedLength, uint16(length))
}
