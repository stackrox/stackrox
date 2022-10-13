package dbhelper

import (
	"bytes"

	"github.com/stackrox/rox/pkg/sliceutils"
)

var (
	separator = []byte("\x00")
)

// GetBucketKey returns a key which combines the prefix and the id with a separator
func GetBucketKey(prefix []byte, id []byte) []byte {
	result := make([]byte, 0, len(prefix)+len(separator)+len(id))
	result = append(result, prefix...)
	result = append(result, separator...)
	result = append(result, id...)
	return result
}

// GetBucketKeyLen returns the length of the bucket
func GetBucketKeyLen(prefix []byte) int {
	return len(prefix) + len(separator)
}

// GetPrefix returns the first prefix found on the input key, and it's remainder afterwards.
func GetPrefix(key []byte) (prefix []byte) {
	idx := bytes.Index(key, separator)
	if idx == -1 {
		return nil
	}
	return sliceutils.ShallowClone(key[:idx])
}

// HasPrefix returns if the given key has the given prefix.
func HasPrefix(prefix []byte, val []byte) bool {
	if len(val) < len(prefix)+len(separator) {
		return false
	}
	return bytes.Equal(prefix, val[:len(prefix)]) && bytes.Equal(separator, val[len(prefix):len(prefix)+len(separator)])
}

// StripPrefix removes a prefix from the val
func StripPrefix(prefix []byte, val []byte) []byte {
	if len(val) >= len(prefix) {
		return val[len(prefix):]
	}
	return val
}

// StripBucket removes a bucket prefix and the separator from the val
func StripBucket(prefix []byte, val []byte) []byte {
	bucket := GetBucketKey(prefix, nil)
	return StripPrefix(bucket, val)
}

// AppendSeparator appends the separator to the end of the key
func AppendSeparator(key []byte) []byte {
	newKey := make([]byte, 0, len(key)+len(separator))
	newKey = append(newKey, key...)
	newKey = append(newKey, separator...)
	return newKey
}

// KV is a key/value pair.
type KV struct {
	Key   []byte
	Value []byte
}
