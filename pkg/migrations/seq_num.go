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
func MinimumSupportedDBVersionSeqNum() int {
	return currentMinimum().Sequence
}

// MinimumSupportedDBVersion is the oldest database version supported
// by the schema at this point in time.
func MinimumSupportedDBVersion() string {
	return currentMinimum().Version
}

func currentMinimum() ReleaseVersion {
	minimum, err := MinimumSupportedForVersion(version.GetMainVersion())
	if err != nil {
		panic(err)
	}
	return minimum
}
