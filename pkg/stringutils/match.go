package stringutils

import "slices"

// MatchesAny returns true if the value matches any of the options
func MatchesAny(value string, options ...string) bool {
	return slices.Contains(options, value)
}
