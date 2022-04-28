package policyversion

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

// Downgrader takes a policy in version N and *must* convert it to N-1.
// It is not expected to perform input version validation inside.
type downgrader func(policy *storage.Policy)

var (
	// If key represents version N, value must be a downgrader from N to N-1.
	// If for version X there is no entry here, X cannot be downgraded.
	// The map must not change during the application runtime.
	// eg: "2.0": downgradeVersion2_0ToVersion1_1 downgrades 2.0 to 1.1
	downgraders = map[string]downgrader{}

	// i-th element is a downgrader from version[i] to version[i-1] or
	// nil, which indicates that downgrade from i to i-1 is impossible.
	// downgradersByVersionRank[0] is always nil.
	downgradersByVersionRank = organizeByVersionRank(downgraders, versions[:])
)

// DowngradePolicyTo attempts to downgrade a given policy to the policy in the
// given target version. Policies in some versions cannot be downgraded.
// The function leaves policy either unchanged or in the downgraded state.
func DowngradePolicyTo(p *storage.Policy, targetVersion PolicyVersion) error {
	currentVersion, err := FromString(p.GetPolicyVersion())
	if err != nil {
		return err
	}

	switch cmp := Compare(currentVersion, targetVersion); {
	case cmp < 0:
		return errors.Errorf("Target version %q is newer than the current policy version %q",
			targetVersion, currentVersion)
	case cmp == 0:
		// No-op
	case cmp > 0:
		// If we can't downgrade all the way to targetVersion, don't even try.
		currentVersionRank := versionRanks[currentVersion.String()]
		targetVersionRank := versionRanks[targetVersion.String()]
		for idx := currentVersionRank; idx > targetVersionRank; idx-- {
			if downgradersByVersionRank[idx] == nil {
				return errors.Errorf("Policy version %q is not downgradable to %q", currentVersion, targetVersion)
			}
		}

		// Downgrade one version at a time.
		for idx := currentVersionRank; idx > currentVersionRank-cmp; idx-- {
			downgradersByVersionRank[idx](p)
		}
	}

	return nil
}

// organizeByVersionRank builds a slice of possibly nil downgraders so that
// versions and their respective downgraders are aligned by their ranks.
func organizeByVersionRank(downgraders map[string]downgrader, versions []string) []downgrader {
	chained := make([]downgrader, 0, len(downgraders))
	for _, version := range versions {
		chained = append(chained, downgraders[version])
	}
	return chained
}
