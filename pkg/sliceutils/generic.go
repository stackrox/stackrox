package sliceutils

import "github.com/mauricelam/genny/generic"

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

//go:generate genny -in=$GOFILE -out=gen-builtins-$GOFILE gen "ElemType=BUILTINS"
