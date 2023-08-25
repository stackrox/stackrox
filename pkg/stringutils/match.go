package stringutils

// MatchesAny returns true if the value matches any of the options
func MatchesAny(value string, options ...string) bool {
	for _, o := range options {
		if o == value {
			return true
		}
	}
	return false
}
