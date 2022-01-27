package sortedkeys

import (
	"bytes"
	"sort"
)

// SortedKeys is a helper class that is a serializable list of keys with a maximum length of 1 << 16. Optimized for a small number of keys stored together.
type SortedKeys [][]byte

type byteSliceSorter [][]byte

func (s byteSliceSorter) Len() int {
	return len(s)
}

func (s byteSliceSorter) Less(i, j int) bool {
	return bytes.Compare(s[i], s[j]) < 0
}

func (s byteSliceSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

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
