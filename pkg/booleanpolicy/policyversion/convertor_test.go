package policyversion

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"gotest.tools/assert"
)

type convertTestCase struct {
	desc     string
	policy   *storage.Policy
	expected *storage.Policy
	hasError bool
}

func TestCloneAndEnsureConverted(t *testing.T) {
	origVersions := versions
	SetupUpgradersForTest(t)
	defer func() {
		versions = origVersions
		upgradersByVersionRank = getUpgradersByVersions(upgraders, versions[:])
	}()

	cases := []convertTestCase{
		{
			desc:     "nil failure",
			policy:   nil,
			expected: nil,
			hasError: true,
		},
		{
			desc: "unknown version",
			policy: &storage.Policy{
				PolicyVersion: "-1",
			},
			expected: nil,
			hasError: true,
		},
		{
			desc:     "Noop when already at current version",
			policy:   &storage.Policy{PolicyVersion: "100.0"},
			expected: &storage.Policy{PolicyVersion: "100.0"},
			hasError: false,
		},
		{
			desc:     "Upgrade from one version below current version",
			policy:   &storage.Policy{PolicyVersion: "99.0"},
			expected: &storage.Policy{PolicyVersion: "100.0"},
			hasError: false,
		},
		{
			desc:     "Upgrade from multiple versions below current version",
			policy:   &storage.Policy{PolicyVersion: "98.0"},
			expected: &storage.Policy{PolicyVersion: "100.0"},
			hasError: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := CloneAndEnsureConverted(tc.policy)
			assert.Assert(t, tc.hasError == (err != nil))
			assert.DeepEqual(t, tc.expected, got)
		})
	}
}
