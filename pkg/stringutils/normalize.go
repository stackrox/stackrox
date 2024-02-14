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
func SanitizeMapValues(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	sanitized := make(map[string]string, len(m))
	for k, v := range m {
		sanitized[k] = sanitizeString(v)
	}
	return sanitized
}
