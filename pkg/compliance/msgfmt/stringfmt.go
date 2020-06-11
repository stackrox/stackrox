package msgfmt

import (
	"fmt"
	"strings"
)

// FormatStrings takes a variadic parameter and returns a consistent interpretation for compliance messages
func FormatStrings(s ...string) string {
	if len(s) == 1 {
		return fmt.Sprintf("%q", s[0])
	}
	return fmt.Sprintf("[ %s ]", strings.Join(s, " | "))
}
