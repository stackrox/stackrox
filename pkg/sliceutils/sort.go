package sliceutils

import (
	"sort"

	"golang.org/x/exp/constraints"
)

type naturallySortableSlice[T constraints.Ordered] []T

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
func NaturalSort[T constraints.Ordered](slice []T) {
	sort.Sort(naturallySortableSlice[T](slice))
}
