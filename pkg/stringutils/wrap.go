package stringutils

import "strings"

// Wrap performs line-wrapping of the given text at 80 characters in length.
// Intended to be used as a template function.
func Wrap(text string) string {
	wrapped := wrapString(text, 80)
	wrapped = strings.TrimSpace(wrapped)
	wrapped = strings.ReplaceAll(wrapped, "\n", "\n      ")
	return wrapped
}

// wrapString wraps text at the given column width, breaking at word boundaries.
func wrapString(s string, limit uint) string {
	var result strings.Builder
	lineLen := 0
	for i, word := range strings.Fields(s) {
		wl := len(word)
		if i > 0 && uint(lineLen+1+wl) > limit {
			result.WriteByte('\n')
			lineLen = 0
		} else if i > 0 {
			result.WriteByte(' ')
			lineLen++
		}
		result.WriteString(word)
		lineLen += wl
	}
	return result.String()
}
