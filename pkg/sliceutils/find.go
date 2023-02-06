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

// FindMatching returns the first index of slice where the passed predicate returns true, or -1 if it doesn't return
// true for any element.
// Example usage:
//
//	FindMatching([]string{"a", "b", "cd"}, func(s string) bool {
//	  return len(s) > 1
//	})
//
// will return 2.
func FindMatching[T any](slice []T, predicate func(T) bool) int {
	for i, v := range slice {
		if predicate(v) {
			return i
		}
	}
	return -1
}
