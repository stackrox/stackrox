package k8srbac

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestFieldSetMerge(t *testing.T) {
	set := NewPolicyRuleFieldSet(CoreFields()...)
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
	set := NewPolicyRuleFieldSet(CoreFields()...)
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
			name: "Same value with names",
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
				ResourceNames: []string{
					"robsPod",
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
				ResourceNames: []string{
					"robsPod",
				},
			},
			expected: true,
		},
		{
			name: "Same value with different names",
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
				ResourceNames: []string{
					"robsPod",
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
				ResourceNames: []string{
					"tomsPod",
				},
			},
			expected: false,
		},
		{
			name: "Same value with missing name",
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
				ResourceNames: []string{
					"tomsPod",
				},
			},
			expected: false,
		},
		{
			name: "Same NonResourceUrl",
			first: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				NonResourceUrls: []string{
					"pods/*",
				},
			},
			second: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				NonResourceUrls: []string{
					"pods/*",
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
	set := NewPolicyRuleFieldSet(CoreFields()...)
	cases := []struct {
		name     string
		first    *storage.PolicyRule
		second   *storage.PolicyRule
		expected bool
	}{
		{
			name: "Matching",
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
			name: "Matching resources but different resource names",
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
				ResourceNames: []string{
					"robsPod",
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
			expected: false,
		},
		{
			name: "Matching resources and second has resource name",
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
				ResourceNames: []string{
					"robsPod",
				},
			},
			expected: true,
		},
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
			name: "Wildcard resource with name",
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
				ResourceNames: []string{
					"derp",
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
			expected: false,
		},
		{
			name: "Wildcard resource with matching name",
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
				ResourceNames: []string{
					"derp",
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
				ResourceNames: []string{
					"derp",
				},
			},
			expected: true,
		},
		{
			name: "Globbed resource url",
			first: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				NonResourceUrls: []string{
					"deployments/*",
					"pods/*",
				},
			},
			second: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				NonResourceUrls: []string{
					"pods/cpus",
				},
			},
			expected: true,
		},
		{
			name: "Globbed deeper resource url",
			first: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				NonResourceUrls: []string{
					"pods/cpus/disks/*",
					"pods/cpus/*",
				},
			},
			second: &storage.PolicyRule{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				NonResourceUrls: []string{
					"pods/cpus", // Needs to be a sub-path to match
				},
			},
			expected: false,
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
