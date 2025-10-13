package stringutils

import (
	"strings"
)

// sanitizeString cleans up invalid characters
func sanitizeString(s string) string {
	s = strings.ToValidUTF8(s, "")
	return strings.ReplaceAll(s, "\x00", "")
}

// SanitizeMapValues cleans up invalid characters from a map to string
func SanitizeMapValues(m map[string]string) {
	for k, v := range m {
		m[k] = sanitizeString(v)
	}
}
