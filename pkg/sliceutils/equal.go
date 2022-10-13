package sliceutils

// Equal checks if the two given slices are equal.
func Equal[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i, aElem := range a {
		if aElem != b[i] {
			return false
		}
	}
	return true
}
