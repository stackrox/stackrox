package booleanpolicy

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPolicySearchIntegration validates that:
// 1. Protobuf fields have the correct search: tags
// 2. Search indexing works properly
func TestPolicySearchIntegration(t *testing.T) {
	testCases := []struct {
		name               string
		fieldName          string
		deploymentModifier func(*storage.Deployment)
		shouldMatch        bool
		policyDescription  string
	}{
		{
			name:              "AllowPrivilegeEscalation: true matches policy",
			fieldName:         fieldnames.AllowPrivilegeEscalation,
			shouldMatch:       true,
			policyDescription: "Container with privilege escalation allowed",
			deploymentModifier: func(d *storage.Deployment) {
				d.Containers[0].SecurityContext = &storage.SecurityContext{
					AllowPrivilegeEscalation: true,
				}
			},
		},
		{
			name:              "AllowPrivilegeEscalation: false does NOT match policy",
			fieldName:         fieldnames.AllowPrivilegeEscalation,
			shouldMatch:       false,
			policyDescription: "Container with privilege escalation allowed",
			deploymentModifier: func(d *storage.Deployment) {
				d.Containers[0].SecurityContext = &storage.SecurityContext{
					AllowPrivilegeEscalation: false, // This was the original bug case!
				}
			},
		},

		{
			name:              "HostNetwork: true matches policy",
			fieldName:         fieldnames.HostNetwork,
			shouldMatch:       true,
			policyDescription: "Host network namespace shared",
			deploymentModifier: func(d *storage.Deployment) {
				d.HostNetwork = true
			},
		},
		{
			name:              "HostNetwork: false does NOT match policy",
			fieldName:         fieldnames.HostNetwork,
			shouldMatch:       false,
			policyDescription: "Host network namespace shared",
			deploymentModifier: func(d *storage.Deployment) {
				d.HostNetwork = false
			},
		},

		{
			name:              "HostPID: true matches policy",
			fieldName:         fieldnames.HostPID,
			shouldMatch:       true,
			policyDescription: "Host PID namespace shared",
			deploymentModifier: func(d *storage.Deployment) {
				d.HostPid = true
			},
		},
		{
			name:              "HostPID: false does NOT match policy",
			fieldName:         fieldnames.HostPID,
			shouldMatch:       false,
			policyDescription: "Host PID namespace shared",
			deploymentModifier: func(d *storage.Deployment) {
				d.HostPid = false
			},
		},

		{
			name:              "HostIPC: true matches policy",
			fieldName:         fieldnames.HostIPC,
			shouldMatch:       true,
			policyDescription: "Host IPC namespace shared",
			deploymentModifier: func(d *storage.Deployment) {
				d.HostIpc = true
			},
		},
		{
			name:              "HostIPC: false does NOT match policy",
			fieldName:         fieldnames.HostIPC,
			shouldMatch:       false,
			policyDescription: "Host IPC namespace shared",
			deploymentModifier: func(d *storage.Deployment) {
				d.HostIpc = false
			},
		},

		{
			name:              "AutomountServiceAccountToken: true matches policy",
			fieldName:         fieldnames.AutomountServiceAccountToken,
			shouldMatch:       true,
			policyDescription: "Service account token automounted",
			deploymentModifier: func(d *storage.Deployment) {
				d.AutomountServiceAccountToken = true
			},
		},
		{
			name:              "AutomountServiceAccountToken: false does NOT match policy",
			fieldName:         fieldnames.AutomountServiceAccountToken,
			shouldMatch:       false,
			policyDescription: "Service account token automounted",
			deploymentModifier: func(d *storage.Deployment) {
				d.AutomountServiceAccountToken = false
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			deployment := fixtures.GetDeployment().CloneVT()
			deployment.Name = tc.name
			if tc.deploymentModifier != nil {
				tc.deploymentModifier(deployment)
			}

			policy := &storage.Policy{
				Name:            tc.policyDescription,
				LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_DEPLOY},
				PolicyVersion:   "1.1",
				PolicySections: []*storage.PolicySection{
					{
						SectionName: "Section 1",
						PolicyGroups: []*storage.PolicyGroup{
							{
								FieldName: tc.fieldName,
								Values: []*storage.PolicyValue{
									{Value: "true"},
								},
							},
						},
					},
				},
			}

			matcher, err := BuildDeploymentMatcher(policy)
			require.NoError(t, err, "Failed to build deployment matcher")

			enhancedDep := EnhancedDeployment{
				Deployment: deployment,
				Images:     fixtures.DeploymentImages(),
				NetworkPoliciesApplied: &augmentedobjs.NetworkPoliciesApplied{
					HasIngressNetworkPolicy: true,
					HasEgressNetworkPolicy:  true,
				},
			}

			violations, err := matcher.MatchDeployment(nil, enhancedDep)
			require.NoError(t, err, "Matcher should not error")

			// Verify match/no-match based on expected behavior
			if tc.shouldMatch {
				assert.NotEmpty(t, violations.AlertViolations,
					"Policy should have matched deployment with %s=true",
					tc.fieldName)
			} else {
				assert.Empty(t, violations.AlertViolations,
					"Policy should NOT have matched deployment with %s=true",
					tc.fieldName)
			}
		})
	}
}
