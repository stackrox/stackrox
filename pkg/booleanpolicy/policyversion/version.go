package policyversion

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	unknownVersionErrMsg = "Unknown policy version"

	// Named policy versions, starting from the most recent. For clarity, it
	// is a good idea to name a version if it is used in checks in the code.

	// version1_1 renamed Policy.whitelists to Policy.exclusions.
	version1_1 = "1.1"

	// version1 introduced PolicySection instead of PolicyFields.
	version1 = "1"

	legacyVersion = ""
)

var (
	// versions enumerates *all* known policy versions and must be in the
	// strictly ascending order.
	versions = [...]string{legacyVersion, version1, version1_1}

	// supportedVersions enumerates all *supported* versions and must be in the
	// strictly ascending order. supportedVersions is _always_ a subset of versions
	// but ordering between versions and supportedVersions is not guaranteed
	supportedVersions = set.NewFrozenStringSet(version1_1)

	// minimumSupportedVersion returns the minimum policy version that is supported.
	// This is calculated and cached separately so that supportedVersions doesn't
	// need to be converted to slice for every comparison.
	minimumSupportedVersion = supportedVersions.AsSlice()[0]

	// versionRanks maps known versions to their sequence numbers. Note that
	// the sequence number may vary among different builds.
	versionRanks = utils.InvertSlice(versions[:])
)

// CurrentVersion is the current version of boolean policies that is handled
// by this package. It shall equal the last element in versions.
func CurrentVersion() PolicyVersion {
	return PolicyVersion{versions[len(versions)-1]}
}

// MinimumSupportedVersion returns the minimum policy version that is supported.
// Anything lower will result in unexpected behavior.
func MinimumSupportedVersion() PolicyVersion {
	return PolicyVersion{minimumSupportedVersion}
}

// IsCurrentVersion returns true if the policyVersion is equal to the current latest version
// Purely a convenient way of using Compare to find if it's equal
func IsCurrentVersion(policyVersion PolicyVersion) bool {
	return Compare(policyVersion, CurrentVersion()) == 0
}

// IsSupportedVersion returns true if the policyVersion is one of the supported versions
// Purely a convenient way of using Compare to find if it's equal
func IsSupportedVersion(policyVersion PolicyVersion) bool {
	stringVer := policyVersion.String()
	return isKnownPolicyVersion(stringVer) && supportedVersions.Contains(stringVer)
}

// PolicyVersion wraps string-based policy version and provides comparison
// operations.
type PolicyVersion struct {
	value string
}

// FromString attempts to convert policy version from a string to the internal
// representation and emits an error in case the version is unknown.
func FromString(policyVersion string) (PolicyVersion, error) {
	if !isKnownPolicyVersion(policyVersion) {
		return PolicyVersion{}, errors.New(unknownVersionErrMsg)
	}
	return PolicyVersion{policyVersion}, nil
}

func (v PolicyVersion) String() string {
	return v.value
}

// Compare returns an integer comparing two valid PolicyVersions.
// The result is essentially a directed distance between a and b.
func Compare(a, b PolicyVersion) int {
	rankA := versionRanks[a.String()]
	rankB := versionRanks[b.String()]

	return rankA - rankB
}

// isKnownPolicyVersion returns true if the supplied string is a known
// policy version.
func isKnownPolicyVersion(version string) bool {
	if _, ok := versionRanks[version]; ok {
		return true
	}
	return false
}
