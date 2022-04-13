package stringutils

// RemoveStringFromSlice will remove the string from the slice.
// If the string is not contained within the slice, the slice will be returned.
func RemoveStringFromSlice(slice []string, s string) []string {
	var res []string
	for _, ss := range slice {
		if s != ss {
			res = append(res, ss)
		}
	}
	return res
}
