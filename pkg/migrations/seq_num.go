package migrations

import (
	"fmt"

	"github.com/stackrox/rox/pkg/migrations/internal"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/productstreams"
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
	minimum, err := MinimumSupportedForVersion(version.GetMainVersion())
	return minimum.Sequence, err
}

// MinimumSupportedDBVersion is the oldest database version supported
// by the schema at this point in time.
func MinimumSupportedDBVersion() (string, error) {
	minimum, err := MinimumSupportedForVersion(version.GetMainVersion())
	return minimum.Version, err
}

// SeqNumForVersion returns the sequence number associated with the
// provided version
func SeqNumForVersion(target productstreams.XYVersion) (int, error) {
	for _, rv := range releaseVersions {
		version, err := productstreams.ParseXYFromVersionString(rv.Version)
		if err != nil {
			return 0, err
		}

		if version.Compare(target) == 0 {
			return rv.Sequence, nil
		}
	}

	return 0, fmt.Errorf("Failed to find sequence number for %q", target)
}
