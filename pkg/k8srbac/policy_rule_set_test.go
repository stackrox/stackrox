package k8srbac

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
)

func TestDeduplicatesPolicyRulesCorrectly(t *testing.T) {
	cases := []struct {
		name     string
		input    []*storage.PolicyRule
		expected []*storage.PolicyRule
	}{
		{
			name: "Same policy rule twice",
			input: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			expected: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
			},
		},
		{
			name: "Different API groups",
			input: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom2",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			expected: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom",
						"custom2",
					},
					Resources: []string{
						"pods",
					},
				},
			},
		},
		{
			name: "Different resources",
			input: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"deployments",
					},
				},
			},
			expected: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"deployments",
						"pods",
					},
				},
			},
		},
		{
			name: "Different verbs",
			input: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
				{
					Verbs: []string{
						"Put",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			expected: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
						"Put",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
			},
		},
		{
			name: "Multiple mixed",
			input: []*storage.PolicyRule{
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"*",
					},
				},
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
				{
					Verbs: []string{
						"Put",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"*",
					},
					Resources: []string{
						"deployments",
					},
				},
				{
					Verbs: []string{
						"Get",
						"Put",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"deployments",
					},
				},
			},
			expected: []*storage.PolicyRule{
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"*",
					},
				},
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"*",
					},
					Resources: []string{
						"deployments",
					},
				},
			},
		},

		{
			name: "boo",
			input: []*storage.PolicyRule{
				{
					Verbs: []string{
						"get",
					},
					ApiGroups: []string{
						"",
					},
					Resources: []string{
						"*",
					},
				},
				{
					Verbs: []string{
						"create",
					},
					ApiGroups: []string{
						"",
					},
					Resources: []string{
						"deployments",
					},
				},
			},
			expected: []*storage.PolicyRule{
				{
					Verbs: []string{
						"get",
					},
					ApiGroups: []string{
						"",
					},
					Resources: []string{
						"*",
					},
				},
				{
					Verbs: []string{
						"create",
					},
					ApiGroups: []string{
						"",
					},
					Resources: []string{
						"deployments",
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prs := NewPolicyRuleSet(APIGroupsField(), ResourcesField(), VerbsField())
			prs.Add(c.input...)
			assert.Equal(t, c.expected, prs.ToSlice())
		})
	}
}

func TestChecksPolicyRuleContentsCorrectly(t *testing.T) {
	cases := []struct {
		name     string
		initial  []*storage.PolicyRule
		grants   *storage.PolicyRule
		expected bool
	}{
		{
			name: "Two different api groups and one matches",
			initial: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom2",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			grants: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			},
			expected: true,
		},
		{
			name: "Different api groups and different resources",
			initial: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"deployments",
					},
				},
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom2",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			grants: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			},
			expected: false,
		},
		{
			name: "Matches one verb in a rule",
			initial: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
						"Put",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"deployments",
					},
				},
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom2",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			grants: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"deployments",
				},
			},
			expected: true,
		},
		{
			name: "Different api group with multiple verbs",
			initial: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
						"Put",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"deployments",
					},
				},
				{
					Verbs: []string{
						"Get",
						"Put",
					},
					ApiGroups: []string{
						"custom2",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			grants: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			},
			expected: false,
		},
		{
			name: "Different multiple resources multiple verbs",
			initial: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
						"Put",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"deployments",
					},
				},
				{
					Verbs: []string{
						"Get",
						"Put",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			grants: &storage.PolicyRule{
				Verbs: []string{
					"Get",
					"Put",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"deployments",
					"pods",
				},
			},
			expected: true,
		},
		{
			name: "Handles verb wildcard",
			initial: []*storage.PolicyRule{
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"deployments",
					},
				},
				{
					Verbs: []string{
						"Get",
						"Put",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			grants: &storage.PolicyRule{
				Verbs: []string{
					"Get",
					"Put",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"deployments",
				},
			},
			expected: true,
		},
		{
			name: "Handles api group wildcard",
			initial: []*storage.PolicyRule{
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"*",
					},
					Resources: []string{
						"deployments",
					},
				},
				{
					Verbs: []string{
						"Get",
						"Put",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			grants: &storage.PolicyRule{
				Verbs: []string{
					"Get",
					"Put",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"deployments",
				},
			},
			expected: true,
		},
		{
			name: "Handles resource wildcard",
			initial: []*storage.PolicyRule{
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"*",
					},
				},
				{
					Verbs: []string{
						"Get",
						"Put",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			grants: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"deployments",
				},
			},
			expected: true,
		},
		{
			name: "Handles multiple wildcards",
			initial: []*storage.PolicyRule{
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"deployments",
					},
				},
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"*",
					},
					Resources: []string{
						"*",
					},
				},
			},
			grants: &storage.PolicyRule{
				Verbs: []string{
					"Get",
					"Put",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			},
			expected: true,
		},
		{
			name: "Doesn't match multiple wildcards",
			initial: []*storage.PolicyRule{
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"deployments",
					},
				},
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"*",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			grants: &storage.PolicyRule{
				Verbs: []string{
					"Get",
					"Put",
				},
				ApiGroups: []string{
					"custom2",
				},
				Resources: []string{
					"deployments",
				},
			},
			expected: false,
		},

		{
			name: "Doesn't match multiple wildcards",
			initial: []*storage.PolicyRule{
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"custom",
					},
					Resources: []string{
						"deployments",
					},
				},
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"*",
					},
					Resources: []string{
						"pods",
					},
				},
			},
			grants: &storage.PolicyRule{
				Verbs: []string{
					"Get",
					"Put",
				},
				ApiGroups: []string{
					"custom2",
				},
				Resources: []string{
					"deployments",
				},
			},
			expected: false,
		},
		{
			name: "rules verbs and resources are not merged",
			initial: []*storage.PolicyRule{
				{
					Verbs: []string{
						"*",
					},
					ApiGroups: []string{
						"",
					},
					Resources: []string{
						"",
					},
				},
				{
					Verbs: []string{
						"get",
					},
					ApiGroups: []string{
						"",
					},
					Resources: []string{
						"*",
					},
				},
			},
			grants:   EffectiveAdmin,
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prs := NewPolicyRuleSet(APIGroupsField(), ResourcesField(), VerbsField())
			prs.Add(c.initial...)
			assert.Equal(t, c.expected, prs.Grants(c.grants))
		})
	}
}

func TestGetPermissionMap(t *testing.T) {
	rules := []*storage.PolicyRule{
		{
			Resources: []string{"foo"},
			Verbs:     []string{"view", "edit"},
		},
		{
			Resources: []string{"*"},
			Verbs:     []string{"edit", "delete"},
		},
		{
			Resources: []string{"bar"},
			Verbs:     []string{"delete", "patch"},
		},
		{
			Resources: []string{"foo"},
			Verbs:     []string{"patch"},
		},
	}
	expected := map[string]set.StringSet{
		"view":   set.NewStringSet("foo"),
		"edit":   set.NewStringSet("*"),
		"delete": set.NewStringSet("*"),
		"patch":  set.NewStringSet("foo", "bar"),
	}
	ruleSet := &policyRuleSet{
		granted: rules,
	}

	permissionMap := ruleSet.GetPermissionMap()
	assert.Equal(t, expected, permissionMap)
}
