package backend

import (
	"errors"
	"regexp"
)

var (
	// ErrInvalidInitBundleName signals that the provided init bundle name contains invalid characters.
	ErrInvalidInitBundleName = errors.New("invalid init bundle name")
	nameValidator            = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

func validateName(s string) error {
	if !nameValidator.MatchString(s) {
		return ErrInvalidInitBundleName
	}
	return nil
}
