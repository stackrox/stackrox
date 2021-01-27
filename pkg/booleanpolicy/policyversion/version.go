package policyversion

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
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

	// versionRanks maps known versions to their sequence numbers. Note that
	// the sequence number may vary among different builds.
	versionRanks = utils.Invert(versions[:]).(map[string]int)
)

// CurrentVersion is the current version of boolean policies that is handled
// by this package. It shall equal the last element in versions.
func CurrentVersion() PolicyVersion {
	return PolicyVersion{versions[len(versions)-1]}
}

// Version1 is the version that introduced PolicySection in favor of
// PolicyFields.
func Version1() PolicyVersion {
	return PolicyVersion{version1}
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

// IsBooleanPolicy returns true if the policy has policy version equal or
// greater to Version1 of boolean policies.
func IsBooleanPolicy(p *storage.Policy) bool {
	v, err := FromString(p.GetPolicyVersion())
	if err != nil {
		return false
	}
	return Compare(v, Version1()) >= 0
}

// isKnownPolicyVersion returns true if the supplied string is a known
// policy version.
func isKnownPolicyVersion(version string) bool {
	if _, ok := versionRanks[version]; ok {
		return true
	}
	return false
}
