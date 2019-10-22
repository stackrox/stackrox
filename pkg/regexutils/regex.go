package regexutils

import (
	"regexp"
)

// MatchWholeString uses a regex and ensures it matches the entire string
func MatchWholeString(r *regexp.Regexp, s string) bool {
	loc := r.FindStringIndex(s)
	if loc == nil {
		return false
	}
	return loc[0] == 0 && loc[1] == len(s)
}
