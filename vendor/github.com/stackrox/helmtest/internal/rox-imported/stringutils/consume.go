package stringutils

import "strings"

// ConsumePrefix checks if *s has the given prefix, and if yes, modifies it
// to remove the prefix. The return value indicates whether the original string
// had the given prefix.
func ConsumePrefix(s *string, prefix string) bool {
	orig := *s
	if !strings.HasPrefix(orig, prefix) {
		return false
	}
	*s = orig[len(prefix):]
	return true
}
