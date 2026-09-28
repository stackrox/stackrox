package set

import "iter"

type Set[KeyType comparable] map[KeyType]struct{}

func (k Set[KeyType]) AsSlice() []KeyType {
	elems := make([]KeyType, 0, len(k))
	for elem := range k {
		elems = append(elems, elem)
	}
	return elems
}

type FrozenSet[KeyType comparable] struct {
	underlying map[KeyType]struct{}
}

func NewFrozenSet[KeyType comparable](elems ...KeyType) FrozenSet[KeyType] {
	m := make(map[KeyType]struct{}, len(elems))
	for _, e := range elems {
		m[e] = struct{}{}
	}
	return FrozenSet[KeyType]{underlying: m}
}

func (k FrozenSet[KeyType]) AsSlice() []KeyType {
	elems := make([]KeyType, 0, len(k.underlying))
	for elem := range k.underlying {
		elems = append(elems, elem)
	}
	return elems
}

func (k FrozenSet[KeyType]) All() iter.Seq[KeyType] {
	return func(yield func(KeyType) bool) {
		for elem := range k.underlying {
			if !yield(elem) {
				return
			}
		}
	}
}

type FrozenStringSet = FrozenSet[string]
type StringSet = Set[string]
