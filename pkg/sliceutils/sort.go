package sliceutils

import (
	"cmp"
	"slices"
	"sort"
)

type naturallySortableSlice[T cmp.Ordered] []T

func (s naturallySortableSlice[T]) Len() int {
	return len(s)
}

func (s naturallySortableSlice[T]) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s naturallySortableSlice[T]) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// NaturalSort sorts the given slice according to the natural ording of elements.
func NaturalSort[T cmp.Ordered](slice []T) {
	sort.Sort(naturallySortableSlice[T](slice))
}

// CopySliceSorted creates a sorted copy of the input slice
func CopySliceSorted[T cmp.Ordered](slice []T) []T {
	sorted := make([]T, len(slice))
	copy(sorted, slice)
	slices.Sort(sorted)
	return sorted
}
