package sliceutils

// Concat concatenates slices, returning a slice with newly allocated backing storage of the exact
// size.
func Concat[T any](slices ...[]T) []T {
	var length int
	for _, slice := range slices {
		length += len(slice)
	}
	result := make([]T, 0, length)
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}
