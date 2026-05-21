package glob

// PatternsOverlap reports whether two glob patterns could match the same path.
func PatternsOverlap(pattern1, pattern2 string) (bool, error) {
	n1, err := buildNFA(pattern1)
	if err != nil {
		return false, err
	}
	n2, err := buildNFA(pattern2)
	if err != nil {
		return false, err
	}

	e1 := eliminateEpsilon(n1)
	e2 := eliminateEpsilon(n2)

	return intersectionNonEmpty(e1, e2), nil
}
