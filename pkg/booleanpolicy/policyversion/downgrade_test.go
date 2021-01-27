package policyversion

import (
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
	noDowngradersFromVersions := set.NewStringSet(legacyVersion, version1)

	for idx := range versions {
		if noDowngradersFromVersions.Contains(versions[idx]) {
			assert.Nil(t, downgradersByVersionRank[idx], "downgrader is not expected for version %q", versions[idx])
		} else {
			assert.NotNil(t, downgradersByVersionRank[idx], "downgrader missing for version %q", versions[idx])
		}
	}
}

func TestDowngradePolicyTo(t *testing.T) {
	type downgradePolicyGoodTestCase struct {
		desc          string
		policy        *storage.Policy
		targetVersion string
		expected      *storage.Policy
	}

	type downgradePolicyBadTestCase struct {
		desc          string
		policy        *storage.Policy
		targetVersion string
	}

	exclusions := []*storage.Exclusion{
		{
			Name: "abcd",
		},
	}

	testCasesGood := []downgradePolicyGoodTestCase{
		{
			"Downgrade from version 1.1 to 1.1",
			&storage.Policy{
				PolicyVersion: version1_1,
				Exclusions:    exclusions,
			},
			version1_1,
			&storage.Policy{
				PolicyVersion: version1_1,
				Exclusions:    exclusions,
			},
		},
		{
			"Downgrade from version 1.1 to 1",
			&storage.Policy{
				PolicyVersion: version1_1,
				Exclusions:    exclusions,
			},
			version1,
			&storage.Policy{
				PolicyVersion: version1,
				Whitelists:    exclusions,
			},
		},
		{
			"Downgrade from version 1 to 1",
			&storage.Policy{
				PolicyVersion: version1,
				Whitelists:    exclusions,
			},
			version1,
			&storage.Policy{
				PolicyVersion: version1,
				Whitelists:    exclusions,
			},
		},
	}

	for _, tc := range testCasesGood {
		t.Run(tc.desc, func(t *testing.T) {
			err := DowngradePolicyTo(tc.policy, PolicyVersion{tc.targetVersion})
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, tc.policy)
		})
	}

	testCasesBad := []downgradePolicyBadTestCase{
		{
			"No downgrade from version 1.1 to legacy",
			&storage.Policy{
				PolicyVersion: version1_1,
			},
			legacyVersion,
		},
		{
			"No downgrade from an unknown version",
			&storage.Policy{
				PolicyVersion: "unknown",
			},
			version1,
		},
		{
			"No downgrade to a newer version",
			&storage.Policy{
				PolicyVersion: version1,
			},
			version1_1,
		},
	}

	for _, tc := range testCasesBad {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Error(t, DowngradePolicyTo(tc.policy, PolicyVersion{tc.targetVersion}))
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
