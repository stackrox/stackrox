package sliceutils

// ReverseInPlace reverses the elements of the given slice in-place.
func ReverseInPlace[T any](slice []T) {
	l := len(slice)
	for i := 0; i < l/2; i++ {
		slice[i], slice[l-1-i] = slice[l-1-i], slice[i]
	}
}

// Reversed returns a slice that contains the elements of the input slice in reverse order.
func Reversed[T any](slice []T) []T {
	out := make([]T, 0, len(slice))
	for i := len(slice) - 1; i >= 0; i-- {
		out = append(out, slice[i])
	}
	return out
}
