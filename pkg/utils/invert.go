package utils

// InvertSlice inverts a slice, returning a map mapping each element in the slice to the _last_ index it occurs at.
func InvertSlice[T comparable](slice []T) map[T]int {
	result := make(map[T]int, len(slice))
	for i, v := range slice {
		result[v] = i
	}
	return result
}

// InvertMap inverts a map, returning a map mapping each value contained in the original map to a key that mapped to
// said value in the original map. In case of multiple keys mapping to a single value in the original map, one key is
// chosen non-deterministically.
func InvertMap[K, V comparable](m map[K]V) map[V]K {
	result := make(map[V]K, len(m))
	for k, v := range m {
		result[v] = k
	}
	return result
}
