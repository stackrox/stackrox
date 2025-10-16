package k8srbac

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
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
			to: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			from: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			mergable: true,
			expected: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
		},
		{
			name: "different resources",
			to: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			from: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"deployments",
				},
			}.Build(),
			mergable: true,
			expected: storage.PolicyRule_builder{
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
			}.Build(),
		},
		{
			name: "different api groups",
			to: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			from: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom2",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			mergable: true,
			expected: storage.PolicyRule_builder{
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
			}.Build(),
		},
		{
			name: "different api groups and resources",
			to: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			from: storage.PolicyRule_builder{
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
			}.Build(),
			mergable: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.mergable, set.Merge(c.to, c.from))
			if c.mergable {
				protoassert.Equal(t, c.expected, c.to)
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
			first: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			second: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			expected: true,
		},
		{
			name: "Same value with names",
			first: storage.PolicyRule_builder{
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
			}.Build(),
			second: storage.PolicyRule_builder{
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
			}.Build(),
			expected: true,
		},
		{
			name: "Same value with different names",
			first: storage.PolicyRule_builder{
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
			}.Build(),
			second: storage.PolicyRule_builder{
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
			}.Build(),
			expected: false,
		},
		{
			name: "Same value with missing name",
			first: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			second: storage.PolicyRule_builder{
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
			}.Build(),
			expected: false,
		},
		{
			name: "Same NonResourceUrl",
			first: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				NonResourceUrls: []string{
					"pods/*",
				},
			}.Build(),
			second: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				NonResourceUrls: []string{
					"pods/*",
				},
			}.Build(),
			expected: true,
		},
		{
			name: "different resources",
			first: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			second: storage.PolicyRule_builder{
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
			}.Build(),
			expected: false,
		},
		{
			name: "different api groups",
			first: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			second: storage.PolicyRule_builder{
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
			}.Build(),
			expected: false,
		},
		{
			name: "different verbs",
			first: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			second: storage.PolicyRule_builder{
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
			}.Build(),
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
			first: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			second: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			expected: true,
		},
		{
			name: "Matching resources but different resource names",
			first: storage.PolicyRule_builder{
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
			}.Build(),
			second: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			expected: false,
		},
		{
			name: "Matching resources and second has resource name",
			first: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			second: storage.PolicyRule_builder{
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
			}.Build(),
			expected: true,
		},
		{
			name: "Wildcarded second",
			first: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			second: storage.PolicyRule_builder{
				Verbs: []string{
					"*",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			expected: false,
		},
		{
			name: "Wildcarded verb and resource",
			first: storage.PolicyRule_builder{
				Verbs: []string{
					"*",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"*",
				},
			}.Build(),
			second: storage.PolicyRule_builder{
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
			}.Build(),
			expected: true,
		},
		{
			name: "Wildcard resource",
			first: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"*",
				},
			}.Build(),
			second: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			expected: true,
		},
		{
			name: "Wildcard resource with name",
			first: storage.PolicyRule_builder{
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
			}.Build(),
			second: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			expected: false,
		},
		{
			name: "Wildcard resource with matching name",
			first: storage.PolicyRule_builder{
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
			}.Build(),
			second: storage.PolicyRule_builder{
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
			}.Build(),
			expected: true,
		},
		{
			name: "Globbed resource url",
			first: storage.PolicyRule_builder{
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
			}.Build(),
			second: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				NonResourceUrls: []string{
					"pods/cpus",
				},
			}.Build(),
			expected: true,
		},
		{
			name: "Globbed deeper resource url",
			first: storage.PolicyRule_builder{
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
			}.Build(),
			second: storage.PolicyRule_builder{
				Verbs: []string{
					"Get",
				},
				ApiGroups: []string{
					"custom",
				},
				NonResourceUrls: []string{
					"pods/cpus", // Needs to be a sub-path to match
				},
			}.Build(),
			expected: false,
		},
		{
			name: "Included verb",
			first: storage.PolicyRule_builder{
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
			}.Build(),
			second: storage.PolicyRule_builder{
				Verbs: []string{
					"Put",
				},
				ApiGroups: []string{
					"custom",
				},
				Resources: []string{
					"pods",
				},
			}.Build(),
			expected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, set.Grants(c.first, c.second))
		})
	}
}
