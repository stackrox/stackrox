package sliceutils

// Unique returns a new slice that contains only the first occurrence of each element in slice.
// Example: Unique([]string{"a", "a", b"}) will return []string{"a", "b"}
func Unique[T comparable](slice []T) []T {
	out := make([]T, 0, len(slice))

	seenElems := make(map[T]struct{})
	for _, elem := range slice {
		preNumElems := len(seenElems)
		seenElems[elem] = struct{}{}
		if len(seenElems) == preNumElems { // not added
			continue
		}
		out = append(out, elem)
	}
	return out
}
