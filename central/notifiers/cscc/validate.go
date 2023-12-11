package cscc

import (
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
)

var (
	validSourceIDPattern = `organizations/[0-9]+/sources/[0-9]+`
	validSourceID        = regexp.MustCompile(validSourceIDPattern)
)

// ValidateSourceID checks the provided SCC Source ID.
func ValidateSourceID(s string) error {
	if !validSourceID.MatchString(s) {
		return errors.Errorf("SCC Source ID must match the format %s", validSourceIDPattern)
	}
	return nil
}

// Validate CSCC notifier.
func Validate(cscc *storage.CSCC, validateSecret bool) error {
	if cscc.SourceId == "" {
		return errors.New("sourceID must be defined in the Cloud SCC Configuration")
	}
	if err := ValidateSourceID(cscc.SourceId); err != nil {
		return err
	}

	if validateSecret && !cscc.GetWifEnabled() && cscc.ServiceAccount == "" {
		return errors.New("serviceAccount must be defined in the Cloud SCC Configuration")
	}

	if cscc.GetWifEnabled() && !features.CloudCredentials.Enabled() {
		return errors.New("cannot use WIF without the feature flag being enabled")
	}
	return nil
}
