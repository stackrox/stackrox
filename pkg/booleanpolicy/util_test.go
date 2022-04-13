package booleanpolicy

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/fieldnames"
	"github.com/stretchr/testify/assert"
)

func TestDiscreteRuntimeSections(t *testing.T) {
	for _, c := range []struct {
		name   string
		policy *storage.Policy
		pass   bool
	}{
		{
			name: "process fields only",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.ProcessName,
							},
						},
					},
				},
			},
			pass: true,
		},
		{
			name: "kubernetes events fields only",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.KubeResource,
							},
						},
					},
				},
			},
			pass: true,
		},
		{
			name: "kubernetes events and process fields in separate sections",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.ProcessName,
							},
						},
					},
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.KubeResource,
							},
						},
					},
				},
			},
			pass: true,
		},
		{
			name: "kubernetes events and process fields in same section",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.ProcessName,
							},
							{
								FieldName: fieldnames.KubeResource,
							},
						},
					},
				},
			},
			pass: false,
		},
		{
			name: "deploy time and runtime fields",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.ImageTag,
							},
							{
								FieldName: fieldnames.KubeResource,
							},
						},
					},
				},
			},
			pass: true,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.pass {
				assert.True(t, ContainsDiscreteRuntimeFieldCategorySections(c.policy))
			} else {
				assert.False(t, ContainsDiscreteRuntimeFieldCategorySections(c.policy))
			}
		})
	}
}
