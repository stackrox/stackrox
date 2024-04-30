package sliceutils

// Map maps the elements of slice, using the given mapFunc
// Example usage:
//
//	Map([]string{"a", "b", "cd"}, func(s string) int {
//	  return len(s)
//	})
//
// will return []int{1, 1, 2}.
func Map[T, U any](slice []T, mapFunc func(T) U) []U {
	result := make([]U, 0, len(slice))
	for _, elem := range slice {
		result = append(result, mapFunc(elem))
	}
	return result
}
