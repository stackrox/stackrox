package errorhelpers

import (
	"fmt"
	"strings"
)

// FormatErrorStrings aggregates a slice of error messages into a single error.
func FormatErrorStrings(start string, errors []string) error {
	if len(errors) > 0 {
		return fmt.Errorf("%s errors: [%s]", start, strings.Join(errors, ", "))
	}
	return nil
}

// FormatErrors aggregates a slice of errors into a single error.
func FormatErrors(start string, errs []error) error {
	var errors []string
	for _, err := range errs {
		errors = append(errors, err.Error())
	}
	return FormatErrorStrings(start, errors)
}
