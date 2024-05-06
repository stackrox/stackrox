package stringutils

import (
	"regexp"
)

// OrDefault returns the string if it's not empty, or the default.
func OrDefault(s, defaultString string) string {
	if s != "" {
		return s
	}
	return defaultString
}

// PointerOrDefault returns the string if it's not nil nor empty, or the default.
func PointerOrDefault(s *string, defaultString string) string {
	if s == nil {
		return defaultString
	}

	return OrDefault(*s, defaultString)
}

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)
