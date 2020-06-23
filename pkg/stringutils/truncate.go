package stringutils

import (
	"fmt"
	"strings"
)

// TruncateOptions are the extra options to be passed when truncating a string
type TruncateOptions interface {
	process(s string) string
}

// WordOriented implies that the cut off will be the end of the last word instead of
// truncating in the middle of a word. If MaxCutOff is greater than zero, this determines
// the maximum number of characters that can be cut off, otherwise we'll relax the
// restriction of cutting off at a word boundary.
type WordOriented struct {
	MaxCutOff int
}

func (w WordOriented) process(s string) string {
	// We'll need to search for a word break not from the end of the string but from three characters before the end of
	// the string, since we're going to append "...".
	if len(s) < 3 {
		// If the length is shorter than 3, we cannot shorten it further since we're always appending '...' at the end.
		return strings.TrimSpace(s)
	}

	idx := strings.LastIndex(s[:len(s)-2], " ")
	if idx == -1 || (w.MaxCutOff > 0 && idx < len(s)-w.MaxCutOff) {
		if len(s) > 3 {
			return fmt.Sprintf("%s...", s[:len(s)-3])
		}
		return s
	}

	trimmed := strings.TrimSpace(s[:idx])
	if trimmed == "" {
		return ""
	}
	return fmt.Sprintf("%s...", strings.TrimSpace(s[:idx]))
}

// Truncate truncates the string if necessary at maxlen.
func Truncate(s string, maxLen int, options ...TruncateOptions) string {
	if len(s) <= maxLen {
		return s
	}
	s = s[:maxLen]
	for _, o := range options {
		s = o.process(s)
	}
	return s
}
