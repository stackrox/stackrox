package compiledpolicies

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestNotAnEnforcementPolicy(t *testing.T) {
	t.Parallel()

	policy, _ := New(&v1.Policy{
		Enforcement: v1.EnforcementAction_UNSET_ENFORCEMENT,
	})
	deployment := &v1.Deployment{
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
	}
	action := v1.ResourceAction_CREATE_RESOURCE
	expectedEnforcement := v1.EnforcementAction_UNSET_ENFORCEMENT
	expectedMessage := ""

	actualEnforcement, actualMsg := policy.GetEnforcementAction(deployment, action)

	assert.Equal(t, expectedEnforcement, actualEnforcement)
	assert.Equal(t, expectedMessage, actualMsg)
}

func TestGlobalService(t *testing.T) {
	t.Parallel()

	policy, _ := New(&v1.Policy{
		Enforcement: v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
	})
	deployment := &v1.Deployment{
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
	}

	action := v1.ResourceAction_CREATE_RESOURCE
	expectedEnforcement := v1.EnforcementAction_UNSET_ENFORCEMENT
	expectedMessage := ""

	actualEnforcement, actualMsg := policy.GetEnforcementAction(deployment, action)

	assert.Equal(t, expectedEnforcement, actualEnforcement)
	assert.Equal(t, expectedMessage, actualMsg)
}

func TestDeamonSet(t *testing.T) {
	t.Parallel()

	policy, _ := New(&v1.Policy{
		Enforcement: v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
	})
	deployment := &v1.Deployment{
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
	}
	action := v1.ResourceAction_CREATE_RESOURCE
	expectedEnforcement := v1.EnforcementAction_UNSET_ENFORCEMENT
	expectedMessage := ""

	actualEnforcement, actualMsg := policy.GetEnforcementAction(deployment, action)

	assert.Equal(t, expectedEnforcement, actualEnforcement)
	assert.Equal(t, expectedMessage, actualMsg)
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	policy, _ := New(&v1.Policy{
		Enforcement: v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
	})

	deployment := &v1.Deployment{
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
	}
	action := v1.ResourceAction_UPDATE_RESOURCE
	expectedEnforcement := v1.EnforcementAction_UNSET_ENFORCEMENT
	expectedMessage := ""

	actualEnforcement, actualMsg := policy.GetEnforcementAction(deployment, action)

	assert.Equal(t, expectedEnforcement, actualEnforcement)
	assert.Equal(t, expectedMessage, actualMsg)
}

func TestScaleToZeroEnforcement(t *testing.T) {
	t.Parallel()

	policy, _ := New(&v1.Policy{
		Enforcement: v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
	})
	deployment := &v1.Deployment{
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
	}
	action := v1.ResourceAction_CREATE_RESOURCE
	expectedEnforcement := v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT
	expectedMessage := "Deployment foobar scaled to 0 replicas in response to policy violation"

	actualEnforcement, actualMsg := policy.GetEnforcementAction(deployment, action)

	assert.Equal(t, expectedEnforcement, actualEnforcement)
	assert.Equal(t, expectedMessage, actualMsg)
}

func TestUnsatisfiableNodeConstraint(t *testing.T) {
	t.Parallel()

	policy, _ := New(&v1.Policy{
		Enforcement: v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT,
	})

	deployment := &v1.Deployment{
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
	}
	action := v1.ResourceAction_CREATE_RESOURCE
	expectedEnforcement := v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT
	expectedMessage := "Unsatisfiable node constraint applied to deployment foobar"

	actualEnforcement, actualMsg := policy.GetEnforcementAction(deployment, action)

	assert.Equal(t, expectedEnforcement, actualEnforcement)
	assert.Equal(t, expectedMessage, actualMsg)
}
