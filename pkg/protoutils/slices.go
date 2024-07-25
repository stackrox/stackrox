package protoutils

// Equalable is an interface for proto objects that have generated Equal method.
type Equalable[T any] interface {
	EqualVT(t T) bool
}

// SliceContains returns whether the given slice of proto objects contains the given proto object.
func SliceContains[T Equalable[T]](msg T, slice []T) bool {
	for _, elem := range slice {
		if elem.EqualVT(msg) {
			return true
		}
	}
	return false
}

// SlicesEqual returns whether the given two slices of proto objects have equal values.
func SlicesEqual[T Equalable[T]](first, second []T) bool {
	if len(first) != len(second) {
		return false
	}
	for i, firstElem := range first {
		secondElem := second[i]
		if !firstElem.EqualVT(secondElem) {
			return false
		}
	}
	return true
}

// SliceUnique returns a slice returning unique values from the given slice.
func SliceUnique[T Equalable[T]](slice []T) []T {
	var uniqueSlice []T
	for _, elem := range slice {
		if !SliceContains(elem, uniqueSlice) {
			uniqueSlice = append(uniqueSlice, elem)
		}
	}
	return uniqueSlice
}
