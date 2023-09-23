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

// FirstNonEmpty returns the first string that is non-empty in the variadic or returns an empty string
func FirstNonEmpty(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return ""
}

// LastNonEmpty returns the last string that is non-empty in the variadic or returns an empty string
func LastNonEmpty(strs ...string) string {
	if len(strs) == 0 {
		return ""
	}
	for i := len(strs) - 1; i >= 0; i-- {
		if strs[i] != "" {
			return strs[i]
		}
	}
	return ""
}
