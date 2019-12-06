package sortedkeys

import (
	"bytes"
	"sort"
)

// SortedKeys is a helper class that is a serializable list of keys with a maximum length of 1 << 16 (maximum supported
// by badger). Optimized for a small number of keys stored together.
type SortedKeys [][]byte

// Sort sorts an input list of keys to create a sorted keys. If you know the keys are already sorted, you can simply
// cast like SortedKeys(keys) instead of using Sort().
func Sort(in [][]byte) SortedKeys {
	ret := make([][]byte, len(in))
	copy(ret, in)
	sort.Slice(ret, func(i, j int) bool {
		return bytes.Compare(ret[i], ret[j]) < 0
	})
	return ret
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

// Remove removes a key from the sorted set.
func (sk SortedKeys) Remove(key []byte) (SortedKeys, bool) {
	pos, exists := sk.positionOf(key)
	if !exists {
		return sk, false
	}
	return sk.removeAt(pos), true
}

// Union combines two sets of sorted keys.
func (sk SortedKeys) Union(other SortedKeys) SortedKeys {
	var maxLen int
	if len(sk) > len(other) {
		maxLen = len(sk)
	} else {
		maxLen = len(other)
	}

	newKeys := make([][]byte, 0, maxLen)
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

// Difference removes all of the keys in other from the received set of keys.
func (sk SortedKeys) Difference(other SortedKeys) SortedKeys {
	newKeys := make([][]byte, 0, len(sk))
	otherIdx := 0
	for _, elem := range sk {
		for otherIdx < len(other) && bytes.Compare(other[otherIdx], elem) < 0 {
			otherIdx++
		}
		if otherIdx >= len(other) || !bytes.Equal(other[otherIdx], elem) {
			newKeys = append(newKeys, elem)
		}
	}
	return newKeys
}

// Intersect creates a new set with only the overlapping keys.
func (sk SortedKeys) Intersect(other SortedKeys) SortedKeys {
	newKeys := make([][]byte, 0, len(sk))
	otherIdx := 0
	thisIdx := 0
	thisInBounds := thisIdx < len(sk)
	otherInBounds := otherIdx < len(other)
	for thisInBounds && otherInBounds {
		cmp := bytes.Compare(sk[thisIdx], other[otherIdx])
		if cmp == 0 {
			newKeys = append(newKeys, sk[thisIdx])
			otherIdx++
			thisIdx++
		} else if cmp > 0 {
			otherIdx++
		} else {
			thisIdx++
		}
		thisInBounds = thisIdx < len(sk)
		otherInBounds = otherIdx < len(other)
	}
	return newKeys
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
	leadingValues := append(append([][]byte{}, sk[:idx]...), key)
	if len(sk) == idx {
		return leadingValues // no values after the insertion index.
	}
	return append(leadingValues, sk[idx:]...)
}

func (sk SortedKeys) removeAt(idx int) SortedKeys {
	leadingValues := append([][]byte{}, sk[:idx]...)
	if len(sk)-1 == idx {
		return leadingValues // no values after the removed index.
	}
	return append(leadingValues, sk[idx+1:]...)
}
