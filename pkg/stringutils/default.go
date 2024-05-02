package stringutils

import (
	"regexp"
	"strings"
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

// ToSnakeCase converts a string from camel case to snake case
func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
