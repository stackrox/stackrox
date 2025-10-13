package sliceutils

import "slices"

// Reversed returns a slice that contains the elements of the input slice in reverse order.
func Reversed[T any](slice []T) []T {
	cloned := slices.Clone(slice)
	slices.Reverse(cloned)
	return cloned
}
