package stringutils

// MatchesAny returns true if the first parameter matches any of the variadic parameters
func MatchesAny(s string, matches ...string) bool {
	for _, m := range matches {
		if s == m {
			return true
		}
	}
	return false
}
