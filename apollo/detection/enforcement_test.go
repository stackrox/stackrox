package detection

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetEnforcementAction(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                string
		policy              *policyWrapper
		deployment          *v1.Deployment
		action              v1.ResourceAction
		expectedEnforcement v1.EnforcementAction
		expectedMessage     string
	}{
		{
			name: "not an enforcement policy",
			policy: &policyWrapper{
				Policy: &v1.Policy{
					Enforce: false,
				},
			},
			deployment: &v1.Deployment{
				Type: "Replicated",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Tag:    "latest",
							Remote: "stackrox/director",
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
			policy: &policyWrapper{
				Policy: &v1.Policy{
					Enforce: true,
				},
			},
			deployment: &v1.Deployment{
				Type: "Global",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Tag:    "latest",
							Remote: "stackrox/director",
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
			policy: &policyWrapper{
				Policy: &v1.Policy{
					Enforce: true,
				},
			},
			deployment: &v1.Deployment{
				Type: "DaemonSet",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Tag:    "latest",
							Remote: "stackrox/director",
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
			policy: &policyWrapper{
				Policy: &v1.Policy{
					Enforce: true,
				},
			},
			deployment: &v1.Deployment{
				Type: "Replicated",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Tag:    "latest",
							Remote: "stackrox/director",
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
			policy: &policyWrapper{
				Policy: &v1.Policy{
					Enforce: true,
				},
			},
			deployment: &v1.Deployment{
				Name: "foobar",
				Type: "Replicated",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Tag:    "latest",
							Remote: "stackrox/director",
						},
					},
				},
			},
			action:              v1.ResourceAction_CREATE_RESOURCE,
			expectedEnforcement: v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
			expectedMessage:     "Deployment foobar scaled to 0 replicas in response to policy violation",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actualEnforcement, actualMsg := c.policy.getEnforcementAction(c.deployment, c.action)

			assert.Equal(t, c.expectedEnforcement, actualEnforcement)
			assert.Equal(t, c.expectedMessage, actualMsg)
		})
	}
}
