package validator

import "github.com/stackrox/rox/generated/api/v1"

// Validator encapsulates the logic for validating license keys, verifying their signatures against a set of registered
// signing keys.
type Validator interface {
	// RegisterSigningKey registers the **PUBLIC** part of a signing key under the given name, with optional
	// restrictions.
	RegisterSigningKey(keyID string, algo string, raw []byte, restrictions SigningKeyRestrictions) error

	// ValidateLicenseKey checks that the given license key is parseable, well-formed, and has a valid signature (including
	// that it complies with the restrictions for the signing key).
	// NOTE: THIS DOES NOT PERFORM ANY CHECKING OF TIMESTAMPS.
	ValidateLicenseKey(licenseKey string) (*v1.License, error)
}
