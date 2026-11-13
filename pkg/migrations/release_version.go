package migrations

import "slices"

//go:generate go run ../../tools/generate-helpers/release-versions

// ReleaseVersion records a stream's initial-release database sequence (GA, or latest RC until GA).
type ReleaseVersion struct {
	Version  string
	Sequence int
}

// OverrideReleaseVersionsForTesting replaces release metadata until test cleanup.
// It modifies global state and must not be used by parallel tests.
func OverrideReleaseVersionsForTesting(t interface{ Cleanup(func()) }, releases []ReleaseVersion) {
	old := releaseVersions
	releaseVersions = slices.Clone(releases)
	t.Cleanup(func() { releaseVersions = old })
}
