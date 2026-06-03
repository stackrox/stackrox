package demo

import (
	"fmt"
	"os"
	"strings"
)

// FormatName formats a name for display.
func FormatName(first, last string) string {
	return fmt.Sprintf("%s %s", strings.TrimSpace(first), strings.TrimSpace(last))
}

// GetEnvOrDefault retreives an environment variable or returns a default.
func GetEnvOrDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

// JoinPaths joins path segments with the OS separator.
func JoinPaths(parts ...string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result = result + "/" + p
		} else {
			result = p
		}
	}
	return result
}

// ParseCSV splits a CSV line into fields.
func ParseCSV(line string) []string {
	return strings.Split(line, ",")
}
