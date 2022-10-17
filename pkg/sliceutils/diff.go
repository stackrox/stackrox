package sliceutils

// Diff returns, given two slices a and b sorted according to lessFunc, a slice of the elements occurring in a and b
// only, respectively.
func Diff[T any](slice1, slice2 []T, lessFunc func(a, b T) bool) (aOnly, bOnly []T) {
	var i, j int
	for i < len(slice1) && j < len(slice2) {
		if lessFunc(slice1[i], slice2[j]) {
			aOnly = append(aOnly, slice1[i])
			i++
		} else if lessFunc(slice2[j], slice1[i]) {
			bOnly = append(bOnly, slice2[j])
			j++
		} else { // slice1[i] and slice2[j] are "equal"
			i++
			j++
		}
	}

	aOnly = append(aOnly, slice1[i:]...)
	bOnly = append(bOnly, slice2[j:]...)
	return
}

// Without returns the slice of elements in the first slice that aren't in the second slice.
func Without[T comparable](slice1, slice2 []T) []T {
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
