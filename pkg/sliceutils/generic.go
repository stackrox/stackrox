package sliceutils

import (
	"github.com/mauricelam/genny/generic"
)

// ElemType is the generic element type of the slice.
type ElemType generic.Type

// ElemTypeDiff returns, given two sorted ElemType slices a and b, a slice of the elements occurring in a and b only,
// respectively.
func ElemTypeDiff(a, b []ElemType, lessFunc func(a, b ElemType) bool) (aOnly, bOnly []ElemType) {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if lessFunc(a[i], b[j]) {
			aOnly = append(aOnly, a[i])
			i++
		} else if lessFunc(b[j], a[i]) {
			bOnly = append(bOnly, b[j])
			j++
		} else { // a[i] and b[j] are "equal"
			i++
			j++
		}
	}

	aOnly = append(aOnly, a[i:]...)
	bOnly = append(bOnly, b[j:]...)
	return
}

// ElemTypeFind returns, given a slice and an element, the first index of elem in the slice, or -1 if the slice does
// not contain elem.
func ElemTypeFind(slice []ElemType, elem ElemType) int {
	for i, sliceElem := range slice {
		if sliceElem == elem {
			return i
		}
	}
	return -1
}

// ElemTypeUnique returns a new slice that contains only the first occurrence of each element in slice.
func ElemTypeUnique(slice []ElemType) []ElemType {
	result := make([]ElemType, 0, len(slice))
	seen := make(map[ElemType]struct{}, len(slice))
	for _, elem := range slice {
		if _, ok := seen[elem]; !ok {
			result = append(result, elem)
			seen[elem] = struct{}{}
		}
	}
	return result
}

// ElemTypeDifference returns the array of elements in the first slice that aren't in the second slice
func ElemTypeDifference(slice1, slice2 []ElemType) []ElemType {
	if len(slice1) == 0 || len(slice2) == 0 {
		return slice1
	}

	blockedElems := make(map[ElemType]struct{}, len(slice2))
	for _, s := range slice2 {
		blockedElems[s] = struct{}{}
	}
	var newSlice []ElemType
	for _, s := range slice1 {
		if _, ok := blockedElems[s]; !ok {
			newSlice = append(newSlice, s)
			blockedElems[s] = struct{}{}
		}
	}
	return newSlice
}

// ElemTypeUnion returns the union array of slice1 and slice2 without duplicates.
// The elements in the returned slice will be in the same order as if you concatenated
// the two slices, and then removed all copies of repeated elements except the first one.
func ElemTypeUnion(slice1, slice2 []ElemType) []ElemType {
	// Fast-path checks
	if len(slice1) == 0 {
		return ElemTypeUnique(slice2)
	}
	if len(slice2) == 0 {
		return ElemTypeUnique(slice1)
	}

	elemSet := make(map[ElemType]struct{}, len(slice1))
	var newSlice []ElemType
	for _, elem := range slice1 {
		if _, ok := elemSet[elem]; !ok {
			elemSet[elem] = struct{}{}
			newSlice = append(newSlice, elem)
		}
	}

	for _, elem := range slice2 {
		if _, ok := elemSet[elem]; !ok {
			elemSet[elem] = struct{}{}
			newSlice = append(newSlice, elem)
		}
	}

	return newSlice
}

// ElemTypeEqual checks if the two given slices are equal.
func ElemTypeEqual(a, b []ElemType) bool {
	if len(a) != len(b) {
		return false
	}
	for i, aElem := range a {
		if aElem != b[i] {
			return false
		}
	}
	return true
}

//go:generate genny -in=$GOFILE -out=gen-builtins-$GOFILE gen "ElemType=BUILTINS"
