package client

import (
	"fmt"
	"regexp"
)

var (
	validSourceIDPattern = `organizations/[0-9]+/sources/[0-9]+`
	validSourceID        = regexp.MustCompile(validSourceIDPattern)
)

// ValidateSourceID checks the provided SCC Source ID for well-formedness.
func ValidateSourceID(s string) error {
	if !validSourceID.MatchString(s) {
		return fmt.Errorf("SCC Source ID must match the format %s", validSourceIDPattern)
	}
	return nil
}
