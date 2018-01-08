package detection

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/detection/processors"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestCompilersInit(t *testing.T) {
	t.Parallel()

	for c := range v1.Policy_Category_name {
		category := v1.Policy_Category(c)

		if category == v1.Policy_Category_UNSET_CATEGORY {
			continue
		}

		if _, ok := processors.PolicyCategoryCompiler[category]; !ok {
			t.Errorf("Policy Compiler not found for %s", category)
		}
	}
}

func TestMatch(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		policy        *v1.Policy
		deployment    *v1.Deployment
		numViolations int
	}{
		{
			name: "latest image tag policy",
			policy: &v1.Policy{
				Name:       "latest",
				Severity:   v1.Severity_LOW_SEVERITY,
				Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
				ImagePolicy: &v1.ImagePolicy{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
				},
			},
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Tag:    "latest",
							Remote: "stackrox/health",
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: false,
						},
					},
				},
			},
			numViolations: 1,
		},
		{
			name: "latest image tag policy and privileged - not privileged",
			policy: &v1.Policy{
				Name:       "latest",
				Severity:   v1.Severity_LOW_SEVERITY,
				Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE, v1.Policy_Category_PRIVILEGES_CAPABILITIES},
				ImagePolicy: &v1.ImagePolicy{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
				},
				PrivilegePolicy: &v1.PrivilegePolicy{
					SetPrivileged: &v1.PrivilegePolicy_Privileged{
						Privileged: true,
					},
				},
			},
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Tag:    "latest",
							Remote: "stackrox/health",
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: false,
						},
					},
				},
			},
			numViolations: 0,
		},
		{
			name: "latest image tag policy and privileged - latest not privileged",
			policy: &v1.Policy{
				Name:       "latest",
				Severity:   v1.Severity_LOW_SEVERITY,
				Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE, v1.Policy_Category_PRIVILEGES_CAPABILITIES},
				ImagePolicy: &v1.ImagePolicy{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
				},
				PrivilegePolicy: &v1.PrivilegePolicy{
					SetPrivileged: &v1.PrivilegePolicy_Privileged{
						Privileged: true,
					},
				},
			},
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Tag:    "latest",
							Remote: "stackrox/health",
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: false,
						},
					},
					{
						Image: &v1.Image{
							Tag:    "1.4",
							Remote: "stackrox/health",
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: true,
						},
					},
				},
			},
			numViolations: 0,
		},
		{
			name: "latest image tag policy and privileged - one match",
			policy: &v1.Policy{
				Name:       "latest",
				Severity:   v1.Severity_LOW_SEVERITY,
				Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE, v1.Policy_Category_PRIVILEGES_CAPABILITIES},
				ImagePolicy: &v1.ImagePolicy{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
				},
				PrivilegePolicy: &v1.PrivilegePolicy{
					SetPrivileged: &v1.PrivilegePolicy_Privileged{
						Privileged: true,
					},
				},
			},
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Tag:    "latest",
							Remote: "stackrox/health",
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: true,
						},
					},
					{
						Image: &v1.Image{
							Tag:    "1.5",
							Remote: "stackrox/zookeeper",
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: true,
						},
					},
				},
			},
			numViolations: 2,
		},
		{
			name: "latest image tag policy and privileged - two matches",
			policy: &v1.Policy{
				Name:       "latest",
				Severity:   v1.Severity_LOW_SEVERITY,
				Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE, v1.Policy_Category_PRIVILEGES_CAPABILITIES},
				ImagePolicy: &v1.ImagePolicy{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
				},
				PrivilegePolicy: &v1.PrivilegePolicy{
					SetPrivileged: &v1.PrivilegePolicy_Privileged{
						Privileged: true,
					},
				},
			},
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Tag:    "latest",
							Remote: "stackrox/health",
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: true,
						},
					},
					{
						Image: &v1.Image{
							Tag:    "latest",
							Remote: "stackrox/zookeeper",
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: true,
						},
					},
				},
			},
			numViolations: 4,
		},
	}

	d := &Detector{}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := newPolicyWrapper(c.policy)

			assert.NoError(t, err)
			assert.NotNil(t, p)

			alert := d.matchPolicy(c.deployment, p)

			if c.numViolations > 0 {
				assert.NotNil(t, alert)
				assert.Equal(t, c.deployment, alert.Deployment)
				assert.Equal(t, c.policy, alert.Policy)
				assert.Equal(t, c.numViolations, len(alert.GetViolations()))
			} else {
				assert.Nil(t, alert)
			}
		})
	}
}
