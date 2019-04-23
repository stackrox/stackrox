package stringutils

// AllEmpty returns true if all the strings that are passed are empty
func AllEmpty(strs ...string) bool {
	for _, s := range strs {
		if s != "" {
			return false
		}
	}
	return true
}

// AllNotEmpty returns true if all the strings that are passed are not empty.
func AllNotEmpty(strs ...string) bool {
	for _, s := range strs {
		if s == "" {
			return false
		}
	}
	return true
}

// AtLeastOneEmpty returns true if at least one of the strings is empty
func AtLeastOneEmpty(strs ...string) bool {
	for _, s := range strs {
		if s == "" {
			return true
		}
	}
	return false
}
