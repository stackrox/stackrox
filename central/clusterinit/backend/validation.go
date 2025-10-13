package backend

import (
	"errors"
	"regexp"
)

var (
	// ErrInvalidInitBundleName signals that the provided init bundle name contains invalid characters.
	ErrInvalidInitBundleName = errors.New("invalid init bundle name")
	// ErrInvalidCRSName signals that the provided CRS name contains invalid characters.
	ErrInvalidCRSName = errors.New("invalid CRS name")
	nameValidator     = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

func validateName(s string) error {
	if !nameValidator.MatchString(s) {
		return ErrInvalidInitBundleName
	}
	return nil
}
