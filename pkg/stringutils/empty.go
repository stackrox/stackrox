package stringutils

import (
	"slices"
)

func isNonEmpty(s string) bool { return s != "" }

// AllEmpty returns true if all the strings that are passed are empty
func AllEmpty(strs ...string) bool {
	return !slices.ContainsFunc(strs, isNonEmpty)
}

// AllNotEmpty returns true if all the strings that are passed are not empty.
func AllNotEmpty(strs ...string) bool {
	return !slices.Contains(strs, "")
}

// AtLeastOneEmpty returns true if at least one of the strings is empty
func AtLeastOneEmpty(strs ...string) bool {
	return slices.Contains(strs, "")
}

// FirstNonEmpty returns the first string that is non-empty in the variadic or returns an empty string
func FirstNonEmpty(strs ...string) string {
	if i := slices.IndexFunc(strs, isNonEmpty); i >= 0 {
		return strs[i]
	}
	return ""
}

// LastNonEmpty returns the last string that is non-empty in the variadic or returns an empty string
func LastNonEmpty(strs ...string) string {
	if len(strs) == 0 {
		return ""
	}
	for _, s := range slices.Backward(strs) {
		if s != "" {
			return s
		}
	}
	return ""
}
