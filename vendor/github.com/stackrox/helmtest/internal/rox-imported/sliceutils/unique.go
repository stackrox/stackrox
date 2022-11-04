package sliceutils

// StringUnique returns a new slice that contains only the first occurrence of each element in slice.
func StringUnique(slice []string) []string {
	result := make([]string, 0, len(slice))
	seen := make(map[string]struct{}, len(slice))
	for _, elem := range slice {
		if _, ok := seen[elem]; !ok {
			result = append(result, elem)
			seen[elem] = struct{}{}
		}
	}
	return result
}
