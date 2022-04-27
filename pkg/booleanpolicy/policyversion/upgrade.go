package policyversion

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

// Upgrader takes a policy in version N and *must* convert it to N+1.
// It is not expected to perform input version validation inside.
type upgrader func(policy *storage.Policy)

var (
	// If key represents version N, value must be a upgrader from N to N+1.
	// If for version X there is no entry here, X cannot be upgraded.
	// The map must not change during the application runtime.
	// eg: "1.1": upgradeVersion1_1ToVersion2_0 upgrades 1.1 to 2.0
	upgraders = map[string]upgrader{}

	// i-th element is an upgrader from version[i] to version[i+1] or
	// nil, which indicates that upgrade from i to i+1 is impossible.
	// the last element is always nil because the latest version cannot be upgraded
	upgradersByVersionRank = getUpgradersByVersions(upgraders, versions[:])
)

// upgradePolicyTo attempts to upgrade a given policy to the policy in the
// given target version. Policies in some versions cannot be upgraded (for e.g. one that is already at the latest).
// The function leaves policy either unchanged or in the upgraded state.
func upgradePolicyTo(p *storage.Policy, targetVersion PolicyVersion) error {
	currentVersion, err := FromString(p.GetPolicyVersion())
	if err != nil {
		return err
	}

	switch cmp := Compare(currentVersion, targetVersion); {
	case cmp > 0:
		return errors.Errorf("Target version %q is older than the current policy version %q",
			targetVersion, currentVersion)
	case cmp == 0:
		// No-op
	case cmp < 0:
		// If we can't upgrade all the way to targetVersion, don't even try.
		currentVersionRank := versionRanks[currentVersion.String()]
		targetVersionRank := versionRanks[targetVersion.String()]
		for idx := currentVersionRank; idx < targetVersionRank; idx++ {
			if upgradersByVersionRank[idx] == nil {
				return errors.Errorf("Policy version %q is not upgradable to %q", currentVersion, targetVersion)
			}
		}

		// Upgrade one version at a time.
		for idx := currentVersionRank; idx < targetVersionRank; idx++ {
			upgradersByVersionRank[idx](p)
		}
	}

	return nil
}

// organizeByVersionRank builds a slice of possibly nil upgraders so that
// versions and their respective upgraders are aligned by their ranks.
func getUpgradersByVersions(upgraders map[string]upgrader, versions []string) []upgrader {
	chained := make([]upgrader, 0, len(upgraders))
	for _, version := range versions {
		chained = append(chained, upgraders[version])
	}
	return chained
}
