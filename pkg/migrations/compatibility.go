package migrations

import (
	"fmt"

	"github.com/stackrox/rox/pkg/version/productstreams"
)

// AllowedUpgradeSkew is the maximum number of release streams in one upgrade.
const AllowedUpgradeSkew = 3

func parseStream(value string) (productstreams.XYVersion, error) {
	xy, err := productstreams.ParseXYFromVersionString(value)
	if err != nil {
		return xy, err
	}
	if xy.X <= 0 || xy.Y < 0 {
		return xy, fmt.Errorf("invalid product version %q", value)
	}
	// Unlike component compatibility, a migration must not accept phantom streams.
	next := productstreams.GetNextYStream(xy)
	previous, err := productstreams.GetPreviousYStream(next)
	if err != nil || previous != xy {
		return xy, fmt.Errorf("unknown product stream %q", value)
	}
	return xy, nil
}

// MinimumSupportedForVersion resolves both minimums from the same release entry.
func MinimumSupportedForVersion(target string) (ReleaseVersion, error) {
	return minimumSupportedForVersion(target, releaseVersions)
}

func minimumSupportedForVersion(target string, releases []ReleaseVersion) (ReleaseVersion, error) {
	floor, err := parseStream(target)
	if err != nil {
		return ReleaseVersion{}, fmt.Errorf("invalid migrator version: %w", err)
	}
	for range AllowedUpgradeSkew {
		floor, err = productstreams.GetPreviousYStream(floor)
		if err != nil {
			return ReleaseVersion{}, fmt.Errorf("finding minimum database version for %q: %w", target, err)
		}
	}
	for _, release := range releases {
		if release.Version == floor.String() {
			return release, nil
		}
	}
	return ReleaseVersion{}, fmt.Errorf("missing initial database sequence for ACS %s (required by %s); run make go-generated-srcs with release tags available", floor, target)
}

// CompatibilityError identifies a permanent startup rejection, not a retryable DB error.
type CompatibilityError struct {
	Message string
}

func (e *CompatibilityError) Error() string { return e.Message }

// CheckRollbackCompatibility retains sequence-based protection understood by old binaries.
func CheckRollbackCompatibility(source MigrationVersion, target string, targetSequence int) error {
	if source.MinimumSeqNum > targetSequence {
		return &CompatibilityError{Message: fmt.Sprintf("Central rollback blocked: database ACS %q requires sequence %d, but ACS %s only supports sequence %d. Use a compatible Central version.", source.MainVersion, source.MinimumSeqNum, target, targetSequence)}
	}
	return nil
}

// CheckUpgradeCompatibility returns the bypassed rejection when an unsafe override
// was necessary. Callers must log that warning. It never overrides rollback safety.
func CheckUpgradeCompatibility(source MigrationVersion, target string, targetSequence int, unsafe bool) (string, error) {
	minimum, err := MinimumSupportedForVersion(target)
	if err != nil {
		return "", err
	}
	if err := CheckRollbackCompatibility(source, target, targetSequence); err != nil {
		return "", err
	}
	if source.SeqNum == 0 && source.MainVersion == "0" && source.MinimumSeqNum == 0 {
		return "", nil
	}
	if source.SeqNum <= 0 || source.MinimumSeqNum < 0 {
		return "", &CompatibilityError{Message: "Central upgrade blocked: invalid database sequence metadata. Contact support before repairing the database."}
	}
	xy, parseErr := parseStream(source.MainVersion)
	var rejection string
	if parseErr != nil {
		rejection = fmt.Sprintf("Central upgrade blocked: cannot determine the source ACS version from %q (database sequence %d) for target ACS %s. Resume the original migrator or contact support for recovery; do not infer a release from the sequence alone.", source.MainVersion, source.SeqNum, target)
	} else {
		floor, _ := parseStream(minimum.Version)
		if xy.Compare(floor) < 0 {
			next := xy
			for range AllowedUpgradeSkew {
				next = productstreams.GetNextYStream(next)
			}
			rejection = fmt.Sprintf("Central upgrade blocked: cannot upgrade database from ACS %s to ACS %s. Minimum supported source: ACS %s. First upgrade through an intermediate Operator release, such as ACS %s, and wait for Central to become ready before retrying. Consult support before downgrading an Operator whose Scanner database has already upgraded.", xy, target, minimum.Version, next)
		}
	}
	if rejection == "" || unsafe {
		return rejection, nil
	}
	return "", &CompatibilityError{Message: rejection}
}
