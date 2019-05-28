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
// truncating in the middle of a word
type WordOriented struct{}

func (WordOriented) process(s string) string {
	if idx := strings.LastIndex(s, " "); idx != -1 {
		trimmed := strings.TrimSpace(s[:idx])
		if trimmed == "" {
			return ""
		}
		return fmt.Sprintf("%s...", strings.TrimSpace(s[:idx]))
	}
	return s
}

// Truncate truncates the string if necessary at maxlen
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
