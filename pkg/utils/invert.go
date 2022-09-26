package utils

func InvertSlice[T comparable](slice []T) map[T]int {
	result := make(map[T]int, len(slice))
	for i, v := range slice {
		result[v] = i
	}
	return result
}

func InvertMap[K, V comparable](m map[K]V) map[V]K {
	result := make(map[V]K, len(m))
	for k, v := range m {
		result[v] = k
	}
	return result
}
