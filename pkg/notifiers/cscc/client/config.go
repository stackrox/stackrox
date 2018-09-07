package client

import (
	"errors"
	"regexp"
)

var (
	allDigits = regexp.MustCompile(`[0-9]+`)
)

// ValidateOrgID checks the provided org ID for well-formedness.
func ValidateOrgID(s string) error {
	if !allDigits.MatchString(s) {
		return errors.New("GCP Org ID must be numeric; note that this is the ID, not the name, of the GCP organization")
	}
	return nil
}
