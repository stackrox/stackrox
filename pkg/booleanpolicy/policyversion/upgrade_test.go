package policyversion

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// If this test fails, chances are you have added a new policy version have not
// provided a upgrader to this new version from the previous one. Consider
// doing so. If it is impossible to upgrade to this version from the previous one, add an
// exception here.
func TestNoUpgraderLeftBehind(t *testing.T) {
	noUpgradersFromVersions := set.NewStringSet(legacyVersion, version1, version1_1)

	for idx := range versions {
		if noUpgradersFromVersions.Contains(versions[idx]) {
			assert.Nil(t, upgradersByVersionRank[idx], "upgrader is not expected for version %q", versions[idx])
		} else {
			assert.NotNil(t, upgradersByVersionRank[idx], "upgrader missing for version %q", versions[idx])
		}
	}
}

func SetupUpgradersForTest(_ *testing.T) {
	simpleUpgrader := func(policy *storage.Policy) {
		v, _ := strconv.ParseFloat(policy.PolicyVersion, 64)
		policy.PolicyVersion = fmt.Sprintf("%.1f", v+1.0)
	}

	// Always set the top three versions to the test versions
	// We know that versions will always have at least three versions so that's a safe number
	ver := 100.0
	for i := len(versions) - 1; i > len(versions)-4; i-- {
		stringVer := fmt.Sprintf("%.1f", ver)

		versions[i] = stringVer
		versionRanks[stringVer] = i
		upgraders[stringVer] = simpleUpgrader
		upgradersByVersionRank[i] = upgraders[stringVer]
		ver--
	}
}

func TestUpgradePolicyTo(t *testing.T) {
	origVersions := versions
	SetupUpgradersForTest(t)
	defer func() {
		versions = origVersions
		upgradersByVersionRank = getUpgradersByVersions(upgraders, versions[:])
	}()

	cases := []struct {
		desc            string
		policy          *storage.Policy
		targetVersion   string
		expectedError   bool
		expectedVersion string
	}{
		{
			"Upgrade from 99 to 100",
			&storage.Policy{
				PolicyVersion: "99.0",
			},
			"100.0",
			false,
			"100.0",
		},
		{
			"Upgrade from 99 to 99 should be no-op",
			&storage.Policy{
				PolicyVersion: "99.0",
			},
			"99.0",
			false,
			"99.0",
		},
		{
			"Upgrade from 99 to an older version should be error",
			&storage.Policy{
				PolicyVersion: "99.0",
			},
			"98.0",
			true,
			"99.0",
		},
		{
			"Upgrade from a non-existent version should be error",
			&storage.Policy{
				PolicyVersion: "199.0",
			},
			"100.0",
			true,
			"199.0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := upgradePolicyTo(tc.policy, PolicyVersion{tc.targetVersion})
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedVersion, tc.policy.GetPolicyVersion())

		})
	}
}

func TestGetUpgradersByVersions(t *testing.T) {
	type organizeByVersionRankTestCase struct {
		desc      string
		upgraders map[string]upgrader
		expected  []upgrader
	}

	dummy := func(p *storage.Policy) {}

	testCases := []organizeByVersionRankTestCase{
		{
			"No upgraders",
			map[string]upgrader{},
			[]upgrader{
				nil, nil, nil,
			},
		},
		{
			"Single upgrader",
			map[string]upgrader{
				version1: dummy,
			},
			[]upgrader{
				nil, dummy, nil,
			},
		},
		{
			"All upgraders",
			map[string]upgrader{
				legacyVersion: dummy,
				version1:      dummy,
				version1_1:    dummy,
			},
			[]upgrader{
				dummy, dummy, dummy,
			},
		},
	}

	versions := []string{legacyVersion, version1, version1_1}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actual := getUpgradersByVersions(tc.upgraders, versions[:])
			require.Len(t, actual, len(versions))

			// Since func values are deeply equal iff both are nil,
			// compare elements one by one.
			for idx := range actual {
				if tc.expected[idx] == nil {
					assert.Nil(t, actual[idx])
				}
			}
		})
	}
}
