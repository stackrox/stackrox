package matcher

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetEnforcementAction(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                string
		policy              *Policy
		deployment          *v1.Deployment
		action              v1.ResourceAction
		expectedEnforcement v1.EnforcementAction
		expectedMessage     string
	}{
		{
			name: "not an enforcement policy",
			policy: &Policy{
				Policy: &v1.Policy{
					Enforcement: v1.EnforcementAction_UNSET_ENFORCEMENT,
				},
			},
			deployment: &v1.Deployment{
				Type: "Replicated",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/director",
							},
						},
					},
				},
			},
			action:              v1.ResourceAction_CREATE_RESOURCE,
			expectedEnforcement: v1.EnforcementAction_UNSET_ENFORCEMENT,
			expectedMessage:     "",
		},
		{
			name: "global service",
			policy: &Policy{
				Policy: &v1.Policy{
					Enforcement: v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				},
			},
			deployment: &v1.Deployment{
				Type: "Global",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/director",
							},
						},
					},
				},
			},
			action:              v1.ResourceAction_CREATE_RESOURCE,
			expectedEnforcement: v1.EnforcementAction_UNSET_ENFORCEMENT,
			expectedMessage:     "",
		},
		{
			name: "daemonset",
			policy: &Policy{
				Policy: &v1.Policy{
					Enforcement: v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				},
			},
			deployment: &v1.Deployment{
				Type: "DaemonSet",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/director",
							},
						},
					},
				},
			},
			action:              v1.ResourceAction_CREATE_RESOURCE,
			expectedEnforcement: v1.EnforcementAction_UNSET_ENFORCEMENT,
			expectedMessage:     "",
		},
		{
			name: "update",
			policy: &Policy{
				Policy: &v1.Policy{
					Enforcement: v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				},
			},
			deployment: &v1.Deployment{
				Type: "Replicated",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/director",
							},
						},
					},
				},
			},
			action:              v1.ResourceAction_UPDATE_RESOURCE,
			expectedEnforcement: v1.EnforcementAction_UNSET_ENFORCEMENT,
			expectedMessage:     "",
		},
		{
			name: "scale to 0 enforcement",
			policy: &Policy{
				Policy: &v1.Policy{
					Enforcement: v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
				},
			},
			deployment: &v1.Deployment{
				Name: "foobar",
				Type: "Replicated",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/director",
							},
						},
					},
				},
			},
			action:              v1.ResourceAction_CREATE_RESOURCE,
			expectedEnforcement: v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
			expectedMessage:     "Deployment foobar scaled to 0 replicas in response to policy violation",
		},
		{
			name: "node constraint enforcement",
			policy: &Policy{
				Policy: &v1.Policy{
					Enforcement: v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
				},
			},
			deployment: &v1.Deployment{
				Name: "foobar",
				Type: "Global",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/director",
							},
						},
					},
				},
			},
			action:              v1.ResourceAction_CREATE_RESOURCE,
			expectedEnforcement: v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
			expectedMessage:     "Unsatisfiable node constraint applied to deployment foobar",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actualEnforcement, actualMsg := c.policy.GetEnforcementAction(c.deployment, c.action)

			assert.Equal(t, c.expectedEnforcement, actualEnforcement)
			assert.Equal(t, c.expectedMessage, actualMsg)
		})
	}
}
