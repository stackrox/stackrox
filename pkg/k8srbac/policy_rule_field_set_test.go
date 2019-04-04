package k8srbac

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestFieldSetMerge(t *testing.T) {
	set := NewPolicyRuleFieldSet(APIGroupsField(), ResourcesField(), VerbsField())
	cases := []struct {
		name     string
		to       *storage.PolicyRule
		from     *storage.PolicyRule
		mergable bool
		expected *storage.PolicyRule
	}{
		{
			name: "Same Value",
			to: &storage.PolicyRule{
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
			from: &storage.PolicyRule{
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
			mergable: true,
			expected: &storage.PolicyRule{
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
		{
			name: "different resources",
			to: &storage.PolicyRule{
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
			from: &storage.PolicyRule{
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
			mergable: true,
			expected: &storage.PolicyRule{
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
		{
			name: "different api groups",
			to: &storage.PolicyRule{
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
			from: &storage.PolicyRule{
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
			mergable: true,
			expected: &storage.PolicyRule{
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
		{
			name: "different api groups and resources",
			to: &storage.PolicyRule{
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
			from: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom2",
				},
				Resources: []string{
					"pods",
					"deployments",
				},
			},
			mergable: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.mergable, set.Merge(c.to, c.from))
			if c.mergable {
				assert.Equal(t, c.expected, c.to)
			}
		})
	}
}

func TestFieldSetEquals(t *testing.T) {
	set := NewPolicyRuleFieldSet(APIGroupsField(), ResourcesField(), VerbsField())
	cases := []struct {
		name     string
		first    *storage.PolicyRule
		second   *storage.PolicyRule
		expected bool
	}{
		{
			name: "Same Value",
			first: &storage.PolicyRule{
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
			second: &storage.PolicyRule{
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
			name: "different resources",
			first: &storage.PolicyRule{
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
			second: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
					"deployments",
				},
			},
			expected: false,
		},
		{
			name: "different api groups",
			first: &storage.PolicyRule{
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
			second: &storage.PolicyRule{
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
			expected: false,
		},
		{
			name: "different verbs",
			first: &storage.PolicyRule{
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
			second: &storage.PolicyRule{
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
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, set.Equals(c.first, c.second))
		})
	}
}

func TestFieldSetGrants(t *testing.T) {
	set := NewPolicyRuleFieldSet(APIGroupsField(), ResourcesField(), VerbsField())
	cases := []struct {
		name     string
		first    *storage.PolicyRule
		second   *storage.PolicyRule
		expected bool
	}{
		{
			name: "Wildcarded second",
			first: &storage.PolicyRule{
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
			second: &storage.PolicyRule{
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
			expected: false,
		},
		{
			name: "Wildcarded verb and resource",
			first: &storage.PolicyRule{
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
			second: &storage.PolicyRule{
				Verbs: []string{
					"Get",
					"Put",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
					"deployments",
				},
			},
			expected: true,
		},
		{
			name: "Wildcard resource",
			first: &storage.PolicyRule{
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
			second: &storage.PolicyRule{
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
			name: "Included verb",
			first: &storage.PolicyRule{
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
			second: &storage.PolicyRule{
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
			expected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, set.Grants(c.first, c.second))
		})
	}
}

func TestFieldSetGranters(t *testing.T) {
	set := NewPolicyRuleFieldSet(VerbsField(), APIGroupsField(), ResourcesField())
	cases := []struct {
		name     string
		input    *storage.PolicyRule
		expected []*storage.PolicyRule
	}{
		{
			name: "Wildcarded second",
			input: &storage.PolicyRule{
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
						"Get",
					},
					ApiGroups: []string{
						"*",
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
						"*",
					},
				},
				{
					Verbs: []string{
						"Get",
					},
					ApiGroups: []string{
						"*",
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
						"*",
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, set.Granters(c.input))
		})
	}
}
