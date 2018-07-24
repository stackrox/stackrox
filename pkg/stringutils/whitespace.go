package stringutils

import (
	"strings"
	"unicode"
)

// ContainsWhitespace returns whether the string contains any whitespace characters.
func ContainsWhitespace(s string) bool {
	return strings.LastIndexFunc(s, unicode.IsSpace) != -1
}
