package sliceutils

// ShallowClone clones a slice, creating a new slice and copying the contents of the underlying array.
// If `in` is a nil slice, a nil slice is returned. If `in` is an empty slice, an empty slice is returned.
func ShallowClone[T any](in []T) []T {
	if in == nil {
		return nil
	}
	if len(in) == 0 {
		return []T{}
	}
	out := make([]T, len(in))
	copy(out, in)
	return out
}
