package policyversion

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDowngradePolicyTo(t *testing.T) {
	type downgradePolicyGoodTestCase struct {
		desc          string
		policy        *storage.Policy
		targetVersion string
		expected      *storage.Policy
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
	}

	for _, tc := range testCasesGood {
		t.Run(tc.desc, func(t *testing.T) {
			err := DowngradePolicyTo(tc.policy, PolicyVersion{tc.targetVersion})
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, tc.policy)
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
				version1_1: dummy,
			},
			[]downgrader{
				dummy, nil, nil,
			},
		},
		{
			"All downgraders",
			map[string]downgrader{
				version1_1: dummy,
			},
			[]downgrader{
				dummy, dummy, dummy,
			},
		},
	}

	versions := []string{version1_1}

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
