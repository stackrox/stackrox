package keys

import (
	"bytes"
	"encoding/base64"

	"github.com/pkg/errors"
)

var sep = []byte(":")

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

// PairKeySelect decodes the given component (0 or 1) from the encoded key.
func PairKeySelect(pairKey []byte, idx int) ([]byte, error) {
	if idx < 0 || idx > 1 {
		return nil, errors.Errorf("invalid pair key index %d, must be 0 or 1", idx)
	}
	parts := bytes.Split(pairKey, sep)
	if len(parts) != 2 {
		return nil, errors.Errorf("invalid pair key %q, expected exactly 2 components after splitting on ':', got %d", pairKey, len(parts))
	}

	part := parts[idx]

	decoded := make([]byte, base64.RawURLEncoding.DecodedLen(len(part)))
	num, err := base64.RawURLEncoding.Decode(decoded, part)
	if err != nil {
		return nil, err
	}
	return decoded[:num], nil
}
