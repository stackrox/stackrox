package validator

import licenseproto "github.com/stackrox/rox/generated/shared/license"

// Validator encapsulates the logic for validating license keys, verifying their signatures against a set of registered
// signing keys.
type Validator interface {
	// RegisterSigningKey registers the **PUBLIC** part of a signing key under the given name, with optional
	// restrictions.
	RegisterSigningKey(algo string, raw []byte, restrictions SigningKeyRestrictions) error

	// ValidateLicenseKey checks that the given license key is parseable, well-formed, and has a valid signature (including
	// that it complies with the restrictions for the signing key).
	// NOTE: THIS DOES NOT PERFORM ANY CHECKING OF TIMESTAMPS.
	ValidateLicenseKey(licenseKey string) (*licenseproto.License, error)
}

// New returns a new validator instance.
func New() Validator {
	return newValidator()
}

//go:generate mockgen-wrapper Validator
