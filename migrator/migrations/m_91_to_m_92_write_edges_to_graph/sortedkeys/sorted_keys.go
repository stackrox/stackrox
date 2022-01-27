package sortedkeys

import (
	"bytes"
	"sort"
)

// SortedKeys is a helper class that is a serializable list of keys with a maximum length of 1 << 16. Optimized for a small number of keys stored together.
type SortedKeys [][]byte

// Find returns the index of the input key, or -1 if it is not found.
func (sk SortedKeys) Find(key []byte) int {
	pos, exists := sk.positionOf(key)
	if !exists {
		return -1
	}
	return pos
}

// Insert adds a key to the sorted set.
func (sk SortedKeys) Insert(key []byte) (SortedKeys, bool) {
	pos, exists := sk.positionOf(key)
	if exists {
		return sk, false
	}
	return sk.insertAt(key, pos), true
}

// Does a binary search for where key fits into the sorted list, returns true in the second param if it is already present.
func (sk SortedKeys) positionOf(key []byte) (int, bool) {
	if len(sk) == 0 {
		return 0, false
	}
	idx := sort.Search(len(sk), func(i int) bool {
		return bytes.Compare(sk[i], key) >= 0
	})
	if idx < len(sk) && bytes.Equal(sk[idx], key) {
		return idx, true
	}
	return idx, false
}

func (sk SortedKeys) insertAt(key []byte, idx int) SortedKeys {
	ret := make([][]byte, 0, len(sk)+1)
	ret = append(append(ret, sk[:idx]...), key)
	if len(sk) == idx {
		return ret // no values after the insertion index.
	}
	return append(ret, sk[idx:]...)
}

// Union combines two sets of sorted keys.
func (sk SortedKeys) Union(other SortedKeys) SortedKeys {
	if len(other) == 0 {
		return CopyKeys(sk)
	}
	newKeys := make([][]byte, 0, len(sk)+len(other))
	otherIdx := 0
	thisIdx := 0
	thisInBounds := thisIdx < len(sk)
	otherInBounds := otherIdx < len(other)
	for thisInBounds || otherInBounds {
		var cmp int
		if thisInBounds && otherInBounds {
			cmp = bytes.Compare(sk[thisIdx], other[otherIdx])
		} else if otherInBounds {
			cmp = 1
		} else {
			cmp = -1
		}

		if cmp == 0 {
			// If they both have the value, add it and move both arrays forward.
			newKeys = append(newKeys, sk[thisIdx])
			otherIdx++
			thisIdx++
		} else if cmp > 0 {
			// Other set has value and not this one, add it and move that one forward.
			newKeys = append(newKeys, other[otherIdx])
			otherIdx++
		} else {
			// This set has value and not the other one, add it and move this one forward.
			newKeys = append(newKeys, sk[thisIdx])
			thisIdx++
		}
		thisInBounds = thisIdx < len(sk)
		otherInBounds = otherIdx < len(other)
	}
	return newKeys
}

// CopyKeys returns a copy of a list of keys.
func CopyKeys(keys [][]byte) [][]byte {
	ret := make([][]byte, len(keys))
	for i := 0; i < len(keys); i++ {
		ret[i] = ByteClone(keys[i])
	}
	return ret
}

// ByteClone clones a slice, creating a new slice
// and copying the contents of the underlying array.
// If `in` is a nil slice, a nil slice is returned.
// If `in` is an empty slice, an empty slice is returned.
func ByteClone(in []byte) []byte {
	if in == nil {
		return nil
	}
	if len(in) == 0 {
		return []byte{}
	}
	out := make([]byte, len(in))
	copy(out, in)
	return out
}
