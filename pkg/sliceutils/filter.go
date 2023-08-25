package sliceutils

// Filter returns a slice containing all elements in `slice` for which the given `filter` func evaluates to true.
func Filter[T any](slice []T, filter func(T) bool) []T {
	var filtered []T
	for _, elem := range slice {
		if filter(elem) {
			filtered = append(filtered, elem)
		}
	}
	return filtered
}
