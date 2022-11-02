package framework

import "regexp"

var (
	whitespaceRegex = regexp.MustCompile(`\s+`)
)

// normalizeString normalizes a given string by replacing all whitespace character segments
// with a single ' ' space.
func normalizeString(s string) string {
	return whitespaceRegex.ReplaceAllString(s, " ")
}
