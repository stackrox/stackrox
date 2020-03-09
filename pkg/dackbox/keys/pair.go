package keys

import (
	"bytes"
	"encoding/base64"
	"fmt"
)

var sep = []byte(":")

// ParsePairKey takes in a key created by CreatePairKey, and returns the two keys used to produce it.
func ParsePairKey(key []byte) ([]byte, []byte, error) {
	k1AndK2 := bytes.Split(key, sep)
	if len(k1AndK2) != 2 {
		return nil, nil, fmt.Errorf("invalid pair id: %s", key)
	}
	decK1Len := base64.RawURLEncoding.DecodedLen(len(k1AndK2[0]))
	k1 := make([]byte, decK1Len)
	n1, err := base64.RawURLEncoding.Decode(k1, k1AndK2[0])
	if err != nil {
		return nil, nil, err
	}
	decK2Len := base64.RawURLEncoding.DecodedLen(len(k1AndK2[1]))
	k2 := make([]byte, decK2Len)
	n2, err := base64.RawURLEncoding.Decode(k2, k1AndK2[1])
	if err != nil {
		return nil, nil, err
	}
	return k1[:n1], k2[:n2], nil
}

// CreatePairKey creates a key from the input pair of keys.
func CreatePairKey(k1, k2 []byte) []byte {
	encK1Len := base64.RawURLEncoding.EncodedLen(len(k1))
	encK2Len := base64.RawURLEncoding.EncodedLen(len(k2))

	ret := make([]byte, encK1Len+encK2Len+len(sep))
	base64.RawURLEncoding.Encode(ret[:encK1Len], k1)
	copy(ret[encK1Len:encK1Len+len(sep)], sep)
	base64.RawURLEncoding.Encode(ret[encK1Len+len(sep):], k2)
	return ret
}
