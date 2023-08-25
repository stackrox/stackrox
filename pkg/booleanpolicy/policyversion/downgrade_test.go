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
// provided a downgrader from this new version to the previous one. Consider
// doing so. If it is impossible to downgrade from the new version, add an
// exception here.
func TestNoDowngraderLeftBehind(t *testing.T) {
	noDowngradersFromVersions := set.NewStringSet(legacyVersion, version1, version1_1)

	for idx := range versions {
		if noDowngradersFromVersions.Contains(versions[idx]) {
			assert.Nil(t, downgradersByVersionRank[idx], "downgrader is not expected for version %q", versions[idx])
		} else {
			assert.NotNil(t, downgradersByVersionRank[idx], "downgrader missing for version %q", versions[idx])
		}
	}
}

func SetupDowngradersForTest(_ *testing.T) {
	simpleDowngrader := func(policy *storage.Policy) {
		v, _ := strconv.ParseFloat(policy.PolicyVersion, 64)
		policy.PolicyVersion = fmt.Sprintf("%.1f", v-1.0)
	}

	// Always set the top three versions to the test versions
	// We know that versions will always have at least three versions so that's a safe number
	ver := 100.0
	for i := len(versions) - 1; i > len(versions)-4; i-- {
		stringVer := fmt.Sprintf("%.1f", ver)

		versions[i] = stringVer
		versionRanks[stringVer] = i
		downgraders[stringVer] = simpleDowngrader
		downgradersByVersionRank[i] = downgraders[stringVer]
		ver--
	}
}

func TestDowngradePolicyTo(t *testing.T) {
	origVersions := versions
	SetupDowngradersForTest(t)
	defer func() {
		versions = origVersions
		downgradersByVersionRank = organizeByVersionRank(downgraders, versions[:])
	}()

	cases := []struct {
		desc            string
		policy          *storage.Policy
		targetVersion   string
		expectedError   bool
		expectedVersion string
	}{
		{
			"Downgrade from 99 to 98",
			&storage.Policy{
				PolicyVersion: "99.0",
			},
			"98.0",
			false,
			"98.0",
		},
		{
			"Downgrade from 99 to 99 should be no-op",
			&storage.Policy{
				PolicyVersion: "99.0",
			},
			"99.0",
			false,
			"99.0",
		},
		{
			"Downgrade from 99 to an newer version should be error",
			&storage.Policy{
				PolicyVersion: "99.0",
			},
			"100.0",
			true,
			"99.0",
		},
		{
			"Downgrade from a non-existent version should be error",
			&storage.Policy{
				PolicyVersion: "199.0",
			},
			"98.0",
			true,
			"199.0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := DowngradePolicyTo(tc.policy, PolicyVersion{tc.targetVersion})
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expectedVersion, tc.policy.GetPolicyVersion())

		})
	}
}

func TestOrganizeByVersionRank(t *testing.T) {
	type organizeByVersionRankTestCase struct {
		desc        string
		downgraders map[string]downgrader
		expected    []downgrader
	}

	dummy := func(p *storage.Policy) {}

	testCases := []organizeByVersionRankTestCase{
		{
			"No downgraders",
			map[string]downgrader{},
			[]downgrader{
				nil, nil, nil,
			},
		},
		{
			"Single downgrader",
			map[string]downgrader{
				version1: dummy,
			},
			[]downgrader{
				nil, dummy, nil,
			},
		},
		{
			"All downgraders",
			map[string]downgrader{
				legacyVersion: dummy,
				version1:      dummy,
				version1_1:    dummy,
			},
			[]downgrader{
				dummy, dummy, dummy,
			},
		},
	}

	versions := []string{legacyVersion, version1, version1_1}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actual := organizeByVersionRank(tc.downgraders, versions[:])
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
