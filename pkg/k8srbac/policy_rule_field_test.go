package k8srbac

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
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
			name: "different Values",
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
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resourcesField.Merge(c.to, c.from)
			assert.Equal(t, c.expected, c.to)
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
			first: &storage.PolicyRule{
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
			second: &storage.PolicyRule{
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
			expected: true,
		},
		{
			name: "Not equal",
			first: &storage.PolicyRule{
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
			second: &storage.PolicyRule{
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
			first: &storage.PolicyRule{
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
			second: &storage.PolicyRule{
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
			expected: true,
		},
		{
			name: "More permissions",
			first: &storage.PolicyRule{
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
			second: &storage.PolicyRule{
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
			name: "Less permissions",
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
					"deployments",
					"pods",
				},
			},
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, resourcesField.Grants(c.first, c.second))
		})
	}
}
