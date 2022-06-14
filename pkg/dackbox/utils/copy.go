package utils

import (
	"github.com/stackrox/stackrox/pkg/sliceutils"
)

// CopyKeys returns a copy of a list of keys.
func CopyKeys(keys [][]byte) [][]byte {
	ret := make([][]byte, len(keys))
	for i := 0; i < len(keys); i++ {
		ret[i] = sliceutils.ByteClone(keys[i])
	}
	return ret
}
