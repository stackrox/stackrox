package migrations

import (
	"github.com/stackrox/rox/pkg/migrations/internal"
	"github.com/stackrox/rox/pkg/version"
)

// CurrentDBVersionSeqNum is the current DB version number.
// This must be incremented every time we write a migration.
// It is a shared constant between central and the migrator binary.
func CurrentDBVersionSeqNum() int {
	return internal.CurrentDBVersionSeqNum
}

// MinimumSupportedDBVersionSeqNum is the oldest database version supported
// by the schema at this point in time.
func MinimumSupportedDBVersionSeqNum() (int, error) {
	minimum, err := currentMinimum()
	return minimum.Sequence, err
}

// MinimumSupportedDBVersion is the oldest database version supported
// by the schema at this point in time.
func MinimumSupportedDBVersion() (string, error) {
	minimum, err := currentMinimum()
	return minimum.Version, err
}

func currentMinimum() (ReleaseVersion, error) {
	return MinimumSupportedForVersion(version.GetMainVersion())
}
