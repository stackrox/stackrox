package sliceutils

import (
	"sort"

	"github.com/mauricelam/genny/generic"
)

// ComparableType is the comparable slice element type.
type ComparableType generic.Number

type sortableComparableTypeSlice []ComparableType

func (s sortableComparableTypeSlice) Len() int {
	return len(s)
}

func (s sortableComparableTypeSlice) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s sortableComparableTypeSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// ComparableTypeSort sorts the given slice.
func ComparableTypeSort(slice []ComparableType) {
	sort.Sort(sortableComparableTypeSlice(slice))
}

//go:generate genny -in=$GOFILE -out=gen-builtins-$GOFILE gen "ComparableType=NUMBERS,string"
