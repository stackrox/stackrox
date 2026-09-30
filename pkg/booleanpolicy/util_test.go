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
			name: "FileAccess only",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.FilePath,
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
			pass: false, // should fail - incompatible runtime categories
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
			name: "Process + FileAccess in same section",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.ProcessName,
							},
							{
								FieldName: fieldnames.FilePath,
							},
						},
					},
				},
			},
			pass: true,
		},
		{
			name: "FileAccess-only section alongside Process+FileAccess section",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.FilePath,
							},
						},
					},
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.ProcessName,
							},
							{
								FieldName: fieldnames.FilePath,
							},
						},
					},
				},
			},
			pass: true,
		},
		{
			name: "Process-only section alongside FileAccess-only section",
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
								FieldName: fieldnames.FilePath,
							},
						},
					},
				},
			},
			pass: false, // should fail - Process section without FileAccess when both are in policy
		},
		{
			name: "Process-only section alongside Process+FileAccess section",
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
								FieldName: fieldnames.ProcessName,
							},
							{
								FieldName: fieldnames.FilePath,
							},
						},
					},
				},
			},
			pass: false, // should fail - Process section without FileAccess when both are in policy
		},
		{
			name: "FileAccess + KubeEvent in same section",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.FilePath,
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
			name: "FileAccess + NetworkFlow across sections",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.FilePath,
							},
						},
					},
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.UnexpectedNetworkFlowDetected,
							},
						},
					},
				},
			},
			pass: false,
		},
		{
			name: "Process + FileAccess + KubeEvent across sections",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.ProcessName,
							},
							{
								FieldName: fieldnames.FilePath,
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
			pass: false,
		},
		{
			name: "NetworkFlow + KubeEvent across sections",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.UnexpectedNetworkFlowDetected,
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
			pass: false,
		},
		{
			name: "Multiple sections each with Process + FileAccess",
			policy: &storage.Policy{
				PolicySections: []*storage.PolicySection{
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.ProcessName,
							},
							{
								FieldName: fieldnames.FilePath,
							},
						},
					},
					{
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: fieldnames.ProcessArguments,
							},
							{
								FieldName: fieldnames.FileOperation,
							},
						},
					},
				},
			},
			pass: true,
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
				assert.True(t, ContainsValidRuntimeFieldCategorySections(c.policy))
			} else {
				assert.False(t, ContainsValidRuntimeFieldCategorySections(c.policy))
			}
		})
	}
}

func TestAuditLogAPIResources(t *testing.T) {
	apiResourcePolicy := func(disabled bool, eventSource storage.EventSource, values ...string) *storage.Policy {
		group := &storage.PolicyGroup{FieldName: fieldnames.KubeAPIResource}
		for _, v := range values {
			group.Values = append(group.Values, &storage.PolicyValue{Value: v})
		}
		return &storage.Policy{
			Disabled:       disabled,
			EventSource:    eventSource,
			PolicySections: []*storage.PolicySection{{PolicyGroups: []*storage.PolicyGroup{group}}},
		}
	}
	enumPolicy := &storage.Policy{
		EventSource: storage.EventSource_AUDIT_LOG_EVENT,
		PolicySections: []*storage.PolicySection{{PolicyGroups: []*storage.PolicyGroup{
			{FieldName: fieldnames.KubeResource, Values: []*storage.PolicyValue{{Value: "SECRETS"}}},
		}}},
	}

	assert.Empty(t, AuditLogAPIResources(nil))
	assert.Empty(t, AuditLogAPIResources([]*storage.Policy{enumPolicy}))
	assert.Equal(t, []string{"applications.argoproj.io", "limitranges", "routes.route.openshift.io"}, AuditLogAPIResources([]*storage.Policy{
		enumPolicy,
		apiResourcePolicy(false, storage.EventSource_AUDIT_LOG_EVENT, "routes.route.openshift.io", "applications.argoproj.io"),
		apiResourcePolicy(false, storage.EventSource_AUDIT_LOG_EVENT, "limitranges", "applications.argoproj.io"),
		apiResourcePolicy(true, storage.EventSource_AUDIT_LOG_EVENT, "issuers.cert-manager.io"),
		apiResourcePolicy(false, storage.EventSource_DEPLOYMENT_EVENT, "nodes"),
	}))
}
