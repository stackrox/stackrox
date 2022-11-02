// Copyright (c) 2020 StackRox Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

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

// ElemTypeClone clones a slice, creating a new slice
// and copying the contents of the underlying array.
// If `in` is a nil slice, a nil slice is returned.
// If `in` is an empty slice, an empty slice is returned.
func ElemTypeClone(in []ElemType) []ElemType {
	if in == nil {
		return nil
	}
	if len(in) == 0 {
		return []ElemType{}
	}
	out := make([]ElemType, len(in))
	copy(out, in)
	return out
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

// ConcatElemTypeSlices concatenates slices, returning a slice with newly allocated backing storage of the exact
// size.
func ConcatElemTypeSlices(slices ...[]ElemType) []ElemType {
	length := 0
	for _, slice := range slices {
		length += len(slice)
	}
	result := make([]ElemType, length)
	i := 0
	for _, slice := range slices {
		nextI := i + len(slice)
		copy(result[i:nextI], slice)
		i = nextI
	}
	return result
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

	slice2Map := make(map[ElemType]struct{}, len(slice2))
	for _, s := range slice2 {
		slice2Map[s] = struct{}{}
	}
	var newSlice []ElemType
	for _, s := range slice1 {
		if _, ok := slice2Map[s]; !ok {
			newSlice = append(newSlice, s)
		}
	}
	return newSlice
}

// ElemTypeUnion returns the union array of slice1 and slice2 without duplicates
func ElemTypeUnion(slice1, slice2 []ElemType) []ElemType {
	// Fast path
	if len(slice1) == 0 {
		return slice2
	}
	if len(slice2) == 0 {
		return slice1
	}

	newSlice := make([]ElemType, len(slice1))
	copy(newSlice, slice1)

	slice1Map := make(map[ElemType]struct{}, len(slice1))
	for _, s := range slice1 {
		slice1Map[s] = struct{}{}
	}
	for _, s := range slice2 {
		if _, ok := slice1Map[s]; !ok {
			newSlice = append(newSlice, s)
		}
	}
	return newSlice
}

//go:generate genny -in=$GOFILE -out=gen-builtins-$GOFILE gen "ElemType=BUILTINS"
