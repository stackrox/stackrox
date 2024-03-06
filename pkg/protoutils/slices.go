package protoutils

import (
	"github.com/stackrox/rox/pkg/protocompat"
)

// SliceContains returns whether the given slice of proto objects contains the given proto object.
func SliceContains[T protocompat.Message](msg T, slice []T) bool {
	for _, elem := range slice {
		if protocompat.Equal(elem, msg) {
			return true
		}
	}
	return false
}

// SlicesEqual returns whether the given two slices of proto objects have equal values.
func SlicesEqual[T protocompat.Message](first, second []T) bool {
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

// SliceUnique returns a slice returning unique values from the given slice.
func SliceUnique[T protocompat.Message](slice []T) []T {
	var uniqueSlice []T
	for _, elem := range slice {
		if !SliceContains(elem, uniqueSlice) {
			uniqueSlice = append(uniqueSlice, elem)
		}
	}
	return uniqueSlice
}
