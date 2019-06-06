package zip

import (
	"regexp"
	"strings"
)

var (
	separators            = regexp.MustCompile(`[ &_=+:/]`)
	notAlphanumericOrDash = regexp.MustCompile(`[^[:alnum:]\-]`)
	dashes                = regexp.MustCompile(`[\-]+`)
)

// GetSafeFilename returns a safe filename for a zip file.
func GetSafeFilename(s string) string {
	// Lowercase to be compatible with all systems. Don't end with a space
	s = strings.ToLower(strings.TrimSpace(s))
	// Replace separators with dash
	s = separators.ReplaceAllString(s, "-")
	// Remove all unknown chars
	s = notAlphanumericOrDash.ReplaceAllString(s, "")
	// multiple dashes to 1 dash
	s = dashes.ReplaceAllString(s, "-")
	return s
}
