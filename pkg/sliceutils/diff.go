package sliceutils

// Diff returns, given two slices a and b sorted according to lessFunc, a slice of the elements occurring in a and b
// only, respectively.
func Diff[T any](a, b []T, lessFunc func(a, b T) bool) (aOnly, bOnly []T) {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if lessFunc(a[i], b[j]) {
			aOnly = append(aOnly, a[i])
			i++
		} else if lessFunc(b[j], a[i]) {
			bOnly = append(bOnly, b[j])
			j++
		} else { // a[i] and b[j] are "equal"
			i++
			j++
		}
	}

	aOnly = append(aOnly, a[i:]...)
	bOnly = append(bOnly, b[j:]...)
	return
}

// Difference returns the array of elements in the first slice that aren't in the second slice
func Difference[T comparable](slice1, slice2 []T) []T {
	if len(slice1) == 0 || len(slice2) == 0 {
		return slice1
	}

	blockedElems := make(map[T]struct{}, len(slice2))
	for _, s := range slice2 {
		blockedElems[s] = struct{}{}
	}
	var newSlice []T
	for _, s := range slice1 {
		if _, ok := blockedElems[s]; !ok {
			newSlice = append(newSlice, s)
			blockedElems[s] = struct{}{}
		}
	}
	return newSlice
}
