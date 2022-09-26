package sliceutils

// Find returns the index of elem in slice, or -1 if slice does not contain elem.
func Find[T comparable](slice []T, elem T) int {
	for i, v := range slice {
		if v == elem {
			return i
		}
	}
	return -1
}

// FindMatching returns the first index of slice where the passed predicate -- which must be a
// func(elemType) bool OR a func(*elemType) bool -- returns true, or -1 if it doesn't return true for any element.
// Example usage:
// FindMatching([]string{"a", "b", "cd"}, func(s string) bool {
//   return len(s) > 1
// })
// will return 2.
// Note that the predicate could also be a func(s *string) bool if you want to avoid copying.
// This function will automatically pass pointers to each slice element if you pass such a predicate.
// It uses reflect, and will be slow.
// It panics at runtime if the arguments are of invalid types. There is no compile-time
// safety of any kind.
// Use ONLY in program initialization blocks, and in tests.
func FindMatching[T any](slice []T, predicate func(T) bool) int {
	for i, v := range slice {
		if predicate(v) {
			return i
		}
	}
	return -1
}
