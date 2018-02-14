package matcher

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/central/detection/processors"
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
		excluded      *v1.DryRunResponse_Excluded
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
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
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
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
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
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: false,
						},
					},
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "1.4",
								Remote: "stackrox/health",
							},
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
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: true,
						},
					},
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "1.5",
								Remote: "stackrox/zookeeper",
							},
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
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: true,
						},
					},
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/zookeeper",
							},
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: true,
						},
					},
				},
			},
			numViolations: 4,
		},
		{
			name: "latest image tag policy with two whitelists that do not match",
			policy: &v1.Policy{
				Name:       "latest",
				Severity:   v1.Severity_LOW_SEVERITY,
				Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
				ImagePolicy: &v1.ImagePolicy{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
				},
				Whitelists: []*v1.Whitelist{
					{
						Container: &v1.Whitelist_Container{
							ImageName: &v1.ImageName{
								Remote: "stackrox/kafka",
							},
						},
					},
					{
						Deployment: &v1.Whitelist_Deployment{
							Scope: &v1.Scope{
								Namespace: "blah",
							},
						},
					},
				},
			},
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
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
			name: "latest image tag policy with two whitelists (2nd matches deployment)",
			policy: &v1.Policy{
				Name:       "latest",
				Severity:   v1.Severity_LOW_SEVERITY,
				Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
				ImagePolicy: &v1.ImagePolicy{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
				},
				Whitelists: []*v1.Whitelist{
					{
						Container: &v1.Whitelist_Container{
							ImageName: &v1.ImageName{
								Remote: "stackrox/kafka",
							},
						},
					},
					{
						Deployment: &v1.Whitelist_Deployment{
							Name: "deployment1",
						},
					},
				},
			},
			deployment: &v1.Deployment{
				Name: "deployment1",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
					},
				},
			},
			excluded: &v1.DryRunResponse_Excluded{
				Deployment: "deployment1",
				Whitelist: &v1.Whitelist{
					Deployment: &v1.Whitelist_Deployment{
						Name: "deployment1",
					},
				},
			},
			numViolations: 0,
		},
		{
			name: "latest image tag policy with two whitelists (2nd matches container)",
			policy: &v1.Policy{
				Name:       "latest",
				Severity:   v1.Severity_LOW_SEVERITY,
				Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
				ImagePolicy: &v1.ImagePolicy{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
				},
				Whitelists: []*v1.Whitelist{
					{
						Container: &v1.Whitelist_Container{
							ImageName: &v1.ImageName{
								Remote: "stackrox/kafka",
							},
						},
					},
					{
						Container: &v1.Whitelist_Container{
							ImageName: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
					},
				},
			},
			deployment: &v1.Deployment{
				Name: "deployment1",
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
					},
				},
			},
			numViolations: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := New(c.policy)

			assert.NoError(t, err)
			assert.NotNil(t, p)

			violations, excluded := p.Match(c.deployment)

			assert.Equal(t, c.numViolations, len(violations))
			assert.Equal(t, c.excluded, excluded)
		})
	}
}

func TestScope(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		policy     *Policy
		deployment *v1.Deployment
		expected   bool
	}{
		{
			name: "disabled",
			policy: &Policy{
				Policy: &v1.Policy{
					Disabled: true,
				},
			},
			deployment: &v1.Deployment{
				ClusterId: "clusterA",
				Namespace: "namespace",
				Labels: map[string]string{
					"key": "value",
					"foo": "bar",
				},
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "wrong cluster",
			policy: &Policy{
				Policy: &v1.Policy{
					Scope: []*v1.Scope{
						{
							Cluster: "clusterB",
						},
					},
				},
			},
			deployment: &v1.Deployment{
				ClusterId: "clusterA",
				Namespace: "namespace",
				Labels: map[string]string{
					"key": "value",
					"foo": "bar",
				},
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "wrong namespace",
			policy: &Policy{
				Policy: &v1.Policy{
					Scope: []*v1.Scope{
						{
							Cluster:   "clusterA",
							Namespace: "notanamespace",
						},
					},
				},
			},
			deployment: &v1.Deployment{
				ClusterId: "clusterA",
				Namespace: "namespace",
				Labels: map[string]string{
					"key": "value",
					"foo": "bar",
				},
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "wrong label",
			policy: &Policy{
				Policy: &v1.Policy{
					Scope: []*v1.Scope{
						{
							Cluster:   "clusterA",
							Namespace: "namespace",
							Label: &v1.Scope_Label{
								Key:   "foo",
								Value: "car",
							},
						},
					},
				},
			},
			deployment: &v1.Deployment{
				ClusterId: "clusterA",
				Namespace: "namespace",
				Labels: map[string]string{
					"key": "value",
					"foo": "bar",
				},
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "match just namespace",
			policy: &Policy{
				Policy: &v1.Policy{
					Scope: []*v1.Scope{
						{
							Namespace: "namespace",
						},
					},
				},
			},
			deployment: &v1.Deployment{
				ClusterId: "clusterA",
				Namespace: "namespace",
				Labels: map[string]string{
					"key": "value",
					"foo": "bar",
				},
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "match all",
			policy: &Policy{
				Policy: &v1.Policy{
					Scope: []*v1.Scope{
						{
							Cluster:   "clusterA",
							Namespace: "namespace",
							Label: &v1.Scope_Label{
								Key:   "foo",
								Value: "bar",
							},
						},
					},
				},
			},
			deployment: &v1.Deployment{
				ClusterId: "clusterA",
				Namespace: "namespace",
				Labels: map[string]string{
					"key": "value",
					"foo": "bar",
				},
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "match one scope",
			policy: &Policy{
				Policy: &v1.Policy{
					Scope: []*v1.Scope{
						{
							Cluster: "clusterA",
						},
						{
							Cluster:   "clusterB",
							Namespace: "namespace",
						},
					},
				},
			},
			deployment: &v1.Deployment{
				ClusterId: "clusterA",
				Namespace: "namespace",
				Labels: map[string]string{
					"key": "value",
					"foo": "bar",
				},
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := c.policy.ShouldProcess(c.deployment)

			assert.Equal(t, c.expected, actual)
		})
	}
}

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
					Enforce: false,
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
					Enforce: true,
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
					Enforce: true,
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
					Enforce: true,
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
					Enforce: true,
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
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actualEnforcement, actualMsg := c.policy.GetEnforcementAction(c.deployment, c.action)

			assert.Equal(t, c.expectedEnforcement, actualEnforcement)
			assert.Equal(t, c.expectedMessage, actualMsg)
		})
	}
}

func TestDeploymentWhitelist(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		expectWhitelist bool
		deployment      *v1.Deployment
		whitelist       *v1.Whitelist
	}{
		{
			name:            "nil whitelist",
			whitelist:       nil,
			deployment:      nil,
			expectWhitelist: false,
		},
		{
			name: "match scope",
			whitelist: &v1.Whitelist{
				Deployment: &v1.Whitelist_Deployment{
					Scope: &v1.Scope{
						Cluster:   "clusterA",
						Namespace: "namespace",
						Label: &v1.Scope_Label{
							Key:   "foo",
							Value: "bar",
						},
					},
				},
			},
			deployment: &v1.Deployment{
				ClusterId: "clusterA",
				Namespace: "namespace",
				Labels: map[string]string{
					"key": "value",
					"foo": "bar",
				},
				Containers: []*v1.Container{
					{
						Image: &v1.Image{
							Name: &v1.ImageName{
								Tag:    "latest",
								Remote: "stackrox/health",
							},
						},
					},
				},
			},
			expectWhitelist: true,
		},
		{
			name: "match service name",
			whitelist: &v1.Whitelist{
				Deployment: &v1.Whitelist_Deployment{
					Name: "deployment1",
				},
			},
			deployment: &v1.Deployment{
				Name: "deployment1",
			},
			expectWhitelist: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := &Policy{}
			assert.Equal(t, c.expectWhitelist, p.matchesDeploymentWhitelist(c.whitelist.GetDeployment(), c.deployment))
		})
	}
}

func TestContainerWhitelist(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		expectWhitelist bool
		container       *v1.Container
		whitelist       *v1.Whitelist
	}{
		{
			name:            "nil whitelist",
			whitelist:       nil,
			container:       nil,
			expectWhitelist: false,
		},
		{
			name: "match registry only",
			whitelist: &v1.Whitelist{
				Container: &v1.Whitelist_Container{
					ImageName: &v1.ImageName{
						Registry: "registry",
					},
				},
			},
			container: &v1.Container{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Registry: "registry",
					},
				},
			},
			expectWhitelist: true,
		},
		{
			name: "match one, but not others",
			whitelist: &v1.Whitelist{
				Container: &v1.Whitelist_Container{
					ImageName: &v1.ImageName{
						Registry: "registry",
						Remote:   "remote",
					},
				},
			},
			container: &v1.Container{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Registry: "registry1",
						Remote:   "remote",
					},
				},
			},
			expectWhitelist: false,
		},
		{
			name: "match all",
			whitelist: &v1.Whitelist{
				Container: &v1.Whitelist_Container{
					ImageName: &v1.ImageName{
						Sha:      "sha",
						Registry: "registry",
						Remote:   "remote",
						Tag:      "tag",
					},
				},
			},
			container: &v1.Container{
				Image: &v1.Image{
					Name: &v1.ImageName{
						Sha:      "sha",
						Registry: "registry",
						Remote:   "remote",
						Tag:      "tag",
					},
				},
			},
			expectWhitelist: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := &Policy{}
			assert.Equal(t, c.expectWhitelist, p.matchesContainerWhitelist(c.whitelist.GetContainer(), c.container))
		})
	}
}
