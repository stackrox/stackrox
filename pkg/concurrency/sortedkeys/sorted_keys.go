package sortedkeys

import (
	"bytes"
	"sort"

	"github.com/stackrox/rox/pkg/sliceutils"
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

// Sort sorts an input list of keys to create a sorted keys. If you know the keys are already sorted, you can simply
// cast like SortedKeys(keys) instead of using Sort().
func Sort(in [][]byte) SortedKeys {
	if len(in) == 0 {
		return in
	}
	ret := make([][]byte, len(in))
	copy(ret, in)
	sort.Sort(byteSliceSorter(ret))
	// dedupe values
	deduped := ret[:1]
	dedIdx := 0
	for retIdx := 1; retIdx < len(ret); retIdx++ {
		if !bytes.Equal(ret[retIdx], deduped[dedIdx]) {
			deduped = append(deduped, ret[retIdx])
			dedIdx++
		}
	}
	return deduped
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
	if len(other) == 0 {
		return sliceutils.ShallowClone2DSlice(sk)
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

// Difference removes all of the keys in other from the received set of keys.
func (sk SortedKeys) Difference(other SortedKeys) SortedKeys {
	if len(other) == 0 {
		return sliceutils.ShallowClone2DSlice(sk)
	}
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
	if len(other) == 0 {
		return nil
	}

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
	ret := make([][]byte, 0, len(sk)+1)
	ret = append(append(ret, sk[:idx]...), key)
	if len(sk) == idx {
		return ret // no values after the insertion index.
	}
	return append(ret, sk[idx:]...)
}

func (sk SortedKeys) removeAt(idx int) SortedKeys {
	ret := make([][]byte, 0, len(sk)-1)
	ret = append(ret, sk[:idx]...)
	if len(sk)-1 == idx {
		return ret // no values after the removed index.
	}
	return append(ret, sk[idx+1:]...)
}

type sortedKeysSliceByFirstElem []SortedKeys

func (s sortedKeysSliceByFirstElem) Len() int {
	return len(s)
}

func (s sortedKeysSliceByFirstElem) Less(i, j int) bool {
	if len(s[i]) == 0 {
		return true
	}
	if len(s[j]) == 0 {
		return false
	}
	return bytes.Compare(s[i][0], s[j][0]) < 0
}

func (s sortedKeysSliceByFirstElem) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// DisjointPrefixUnion returns the union of the given sets, assuming that each set has a distinct prefix.
func DisjointPrefixUnion(skss ...SortedKeys) SortedKeys {
	sort.Sort(sortedKeysSliceByFirstElem(skss))
	total := 0
	for _, sks := range skss {
		total += len(sks)
	}
	result := make(SortedKeys, 0, total)
	for _, sks := range skss {
		result = append(result, sks...)
	}
	return result
}
