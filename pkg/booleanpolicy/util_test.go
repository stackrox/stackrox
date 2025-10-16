package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
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
			policy: storage.Policy_builder{
				PolicySections: []*storage.PolicySection{
					storage.PolicySection_builder{
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: fieldnames.ProcessName,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
			pass: true,
		},
		{
			name: "kubernetes events fields only",
			policy: storage.Policy_builder{
				PolicySections: []*storage.PolicySection{
					storage.PolicySection_builder{
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: fieldnames.KubeResource,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
			pass: true,
		},
		{
			name: "kubernetes events and process fields in separate sections",
			policy: storage.Policy_builder{
				PolicySections: []*storage.PolicySection{
					storage.PolicySection_builder{
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: fieldnames.ProcessName,
							}.Build(),
						},
					}.Build(),
					storage.PolicySection_builder{
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: fieldnames.KubeResource,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
			pass: true,
		},
		{
			name: "kubernetes events and process fields in same section",
			policy: storage.Policy_builder{
				PolicySections: []*storage.PolicySection{
					storage.PolicySection_builder{
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: fieldnames.ProcessName,
							}.Build(),
							storage.PolicyGroup_builder{
								FieldName: fieldnames.KubeResource,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
			pass: false,
		},
		{
			name: "deploy time and runtime fields",
			policy: storage.Policy_builder{
				PolicySections: []*storage.PolicySection{
					storage.PolicySection_builder{
						PolicyGroups: []*storage.PolicyGroup{
							storage.PolicyGroup_builder{
								FieldName: fieldnames.ImageTag,
							}.Build(),
							storage.PolicyGroup_builder{
								FieldName: fieldnames.KubeResource,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
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
