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

// MapsIntersect returns true if there is at least one key-value pair that is present in both maps
// If both, or either maps are empty, it returns false
// TODO : Convert to generics after upgrade to go 1.18
func MapsIntersect[K, V comparable](m1 map[K]V, m2 map[K]V) bool {
	if len(m2) == 0 {
		return false
	}
	if len(m1) > len(m2) {
		// Range over smaller map
		m1, m2 = m2, m1
	}
	for k, v := range m1 {
		if val, exists := m2[k]; exists {
			if v == val {
				return true
			}
		}
	}
	return false
}
