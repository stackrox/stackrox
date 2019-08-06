package stringutils

import (
	"strings"
)

// EnsureSuffix ensures that the string ends with the given suffix,
// adding the suffix IF AND ONLY IF it's not present already.
func EnsureSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		return s
	}
	return s + suffix
}
