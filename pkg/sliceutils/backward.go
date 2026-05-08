package sliceutils

import (
	"iter"
	"slices"
)

// Backward returns an iterator over index-value pairs in the slice,
// traversing it backward with descending indices.
func Backward[Slice ~[]E, E any](s Slice) iter.Seq2[int, E] {
	return slices.Backward(s)
}
