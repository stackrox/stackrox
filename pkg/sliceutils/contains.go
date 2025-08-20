package sliceutils

// Contains returns true if value is present in the slice, false otherwise
func Contains[T comparable](slice []T, value T) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
