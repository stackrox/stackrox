package policyversion

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

type convertTestCase struct {
	desc     string
	policy   *storage.Policy
	expected *storage.Policy
	hasError bool
}

func TestCloneAndEnsureConverted(t *testing.T) {
	fields := &storage.PolicyFields{
		Cvss: &storage.NumericalPolicy{
			Op:    storage.Comparator_GREATER_THAN_OR_EQUALS,
			Value: 7.0,
		},
	}
	sections := []*storage.PolicySection{
		{
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: fieldnames.CVSS,
					Values: []*storage.PolicyValue{
						{
							Value: ">= 7.000000",
						},
					},
				},
			},
		},
	}
	exclusions := []*storage.Exclusion{
		{
			Name: "abcd",
		},
	}

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
			desc: "empty sections",
			policy: &storage.Policy{
				PolicyVersion: Version1().String(),
			},
			expected: nil,
			hasError: true,
		},
		{
			desc: "empty fields",
			policy: &storage.Policy{
				PolicyVersion: legacyVersion,
			},
			expected: nil,
			hasError: true,
		},
		{
			desc: "whitelists in version greater than 1",
			policy: &storage.Policy{
				PolicyVersion:  CurrentVersion().String(),
				PolicySections: sections,
				Whitelists:     exclusions,
			},
			expected: nil,
			hasError: true,
		},
		{
			desc: "both whitelists and exclusions",
			policy: &storage.Policy{
				PolicyVersion:  Version1().String(),
				PolicySections: sections,
				Whitelists:     exclusions,
				Exclusions:     exclusions,
			},
			expected: nil,
			hasError: true,
		},
		{
			desc: "valid conversion",
			policy: &storage.Policy{
				Fields: fields,
			},
			expected: &storage.Policy{
				PolicyVersion:  CurrentVersion().String(),
				PolicySections: sections,
			},
		},
		{
			desc: "valid conversion with legacy version",
			policy: &storage.Policy{
				PolicyVersion: legacyVersion,
				Fields:        fields,
			},
			expected: &storage.Policy{
				PolicyVersion:  CurrentVersion().String(),
				PolicySections: sections,
			},
		},
		{
			desc: "valid conversion with legacy version and whitelists",
			policy: &storage.Policy{
				PolicyVersion: legacyVersion,
				Fields:        fields,
				Whitelists:    exclusions,
			},
			expected: &storage.Policy{
				PolicyVersion:  CurrentVersion().String(),
				PolicySections: sections,
				Exclusions:     exclusions,
			},
		},
		{
			desc: "valid conversion with version 1 and whitelists",
			policy: &storage.Policy{
				PolicyVersion:  Version1().String(),
				PolicySections: sections,
				Whitelists:     exclusions,
			},
			expected: &storage.Policy{
				PolicyVersion:  CurrentVersion().String(),
				PolicySections: sections,
				Exclusions:     exclusions,
			},
		},
		{
			desc: "valid noop with sections",
			policy: &storage.Policy{
				PolicyVersion:  Version1().String(),
				PolicySections: sections,
			},
			expected: &storage.Policy{
				PolicyVersion:  CurrentVersion().String(),
				PolicySections: sections,
			},
		},
		{
			desc: "valid noop with empty exclusions",
			policy: &storage.Policy{
				PolicyVersion:  Version1().String(),
				PolicySections: sections,
			},
			expected: &storage.Policy{
				PolicyVersion:  CurrentVersion().String(),
				PolicySections: sections,
			},
		},
		{
			desc: "valid noop with exclusions",
			policy: &storage.Policy{
				PolicyVersion:  Version1().String(),
				PolicySections: sections,
				Exclusions:     exclusions,
			},
			expected: &storage.Policy{
				PolicyVersion:  CurrentVersion().String(),
				PolicySections: sections,
				Exclusions:     exclusions,
			},
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

func TestMigrateLegacyPolicy(t *testing.T) {
	mockExclusion := &storage.Exclusion{
		Name: "abcd",
		Image: &storage.Exclusion_Image{
			Name: "some name",
		},
	}
	mockScope := &storage.Scope{
		Label: &storage.Scope_Label{
			Key:   "Joseph",
			Value: "Rules",
		},
	}

	legacyPolicy := &storage.Policy{
		Id:              "Some ID",
		Name:            "Some Name",
		Description:     "Some Description",
		LifecycleStages: nil,
		Exclusions: []*storage.Exclusion{
			mockExclusion,
		},
		Scope: []*storage.Scope{
			mockScope,
		},
		Fields: &storage.PolicyFields{
			ImageName: &storage.ImageNamePolicy{
				Registry: "123",
				Remote:   "456",
				Tag:      "789",
			},
		},
	}
	expectedSections := []*storage.PolicySection{
		{
			PolicyGroups: []*storage.PolicyGroup{
				{
					FieldName: fieldnames.ImageRegistry,
					Values: []*storage.PolicyValue{
						{
							Value: "123",
						},
					},
				},
				{
					FieldName: fieldnames.ImageRemote,
					Values: []*storage.PolicyValue{
						{
							Value: "r/.*456.*",
						},
					},
				},
				{
					FieldName: fieldnames.ImageTag,
					Values: []*storage.PolicyValue{
						{
							Value: "789",
						},
					},
				},
			},
		},
	}

	t.Run("test migrator", func(t *testing.T) {
		booleanPolicy, err := CloneAndEnsureConverted(legacyPolicy)
		require.NoError(t, err)
		require.Equal(t, CurrentVersion().String(), booleanPolicy.GetPolicyVersion())
		require.Equal(t, expectedSections, booleanPolicy.GetPolicySections())
	})
}
