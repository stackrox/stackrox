package stringutils

import "strings"

// MaybeTrimPrefix trims the prefix from the given input string, if it exists as a prefix.
func MaybeTrimPrefix(s, prefix string) (string, bool) {
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix):], true
	}
	return s, false
}
