package stringutils

import (
	"fmt"
	"strings"
)

// WriteStrings writes the given strings to the builder, and ignores the error. (The documentation for Builder says it always
// returns a nil error.)
func WriteStrings(sb *strings.Builder, strings ...string) {
	for _, s := range strings {
		_, _ = sb.WriteString(s)
	}
}

// WriteStringf writes the given formatted string to the builder, and ignores the error.
func WriteStringf(sb *strings.Builder, format string, args ...interface{}) {
	_, _ = sb.WriteString(fmt.Sprintf(format, args...))
}
