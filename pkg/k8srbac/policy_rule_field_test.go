package k8srbac

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestFieldMerge(t *testing.T) {
	resourcesField := ResourcesField()
	cases := []struct {
		name     string
		to       *storage.PolicyRule
		from     *storage.PolicyRule
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
			name: "different Values",
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
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resourcesField.Merge(c.to, c.from)
			protoassert.Equal(t, c.expected, c.to)
		})
	}
}

func TestFieldEquals(t *testing.T) {
	resourcesField := ResourcesField()
	cases := []struct {
		name     string
		first    *storage.PolicyRule
		second   *storage.PolicyRule
		expected bool
	}{
		{
			name: "Equal",
			first: storage.PolicyRule_builder{
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
			second: storage.PolicyRule_builder{
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
			expected: true,
		},
		{
			name: "Not equal",
			first: storage.PolicyRule_builder{
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
			second: storage.PolicyRule_builder{
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
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, resourcesField.Equals(c.first, c.second))
		})
	}
}

func TestFieldGrants(t *testing.T) {
	resourcesField := ResourcesField()
	cases := []struct {
		name     string
		first    *storage.PolicyRule
		second   *storage.PolicyRule
		expected bool
	}{
		{
			name: "Same permissions",
			first: storage.PolicyRule_builder{
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
			second: storage.PolicyRule_builder{
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
			expected: true,
		},
		{
			name: "More permissions",
			first: storage.PolicyRule_builder{
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
			second: storage.PolicyRule_builder{
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
			expected: true,
		},
		{
			name: "Less permissions",
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
					"deployments",
					"pods",
				},
			}.Build(),
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, resourcesField.Grants(c.first, c.second))
		})
	}
}
