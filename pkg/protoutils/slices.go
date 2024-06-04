package protoutils

import (
	"github.com/mitchellh/hashstructure/v2"
	"github.com/stackrox/rox/pkg/protocompat"
)

// SliceContains returns whether the given slice of proto objects contains the given proto object.
func SliceContains[T protocompat.Equalable[T]](msg T, slice []T) bool {
	for _, elem := range slice {
		if protocompat.Equal(elem, msg) {
			return true
		}
	}
	return false
}

// SlicesEqual returns whether the given two slices of proto objects have equal values.
func SlicesEqual[T protocompat.Equalable[T]](first, second []T) bool {
	if len(first) != len(second) {
		return false
	}
	for i, firstElem := range first {
		secondElem := second[i]
		if !protocompat.Equal(firstElem, secondElem) {
			return false
		}
	}
	return true
}

// SlicesEqualValues returns whether the given two slices of proto objects
// have equal values irrelevant of order.
// NOTE: This is dummy implementation, just to check Postgres unit tests.
func SlicesEqualValues[T protocompat.Message](first, second []T) bool {
	if len(first) != len(second) {
		return false
	}

	xorFirst := uint64(0)
	xorSecond := uint64(0)
	for i := 0; i < len(first); i++ {
		firstHash, err := hashstructure.Hash(first[i], hashstructure.FormatV2, &hashstructure.HashOptions{ZeroNil: true})
		if err != nil {
			return false
		}
		xorFirst ^= firstHash

		secondHash, err := hashstructure.Hash(second[i], hashstructure.FormatV2, &hashstructure.HashOptions{ZeroNil: true})
		if err != nil {
			return false
		}
		xorSecond ^= secondHash
	}

	return xorFirst == xorSecond
}

// SliceUnique returns a slice returning unique values from the given slice.
func SliceUnique[T protocompat.Equalable[T]](slice []T) []T {
	var uniqueSlice []T
	for _, elem := range slice {
		if !SliceContains(elem, uniqueSlice) {
			uniqueSlice = append(uniqueSlice, elem)
		}
	}
	return uniqueSlice
}
