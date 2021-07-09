package keys

import (
	"encoding/base64"
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
