package matcher

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

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
				Categories: []string{"Image Assurance"},
				Fields: &v1.PolicyFields{
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
				Categories: []string{"Image Assurance", "Privileges Capabilities"},
				Fields: &v1.PolicyFields{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
					SetPrivileged: &v1.PolicyFields_Privileged{
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
				Categories: []string{"Image Assurance", "Privileges Capabilities"},
				Fields: &v1.PolicyFields{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
					SetPrivileged: &v1.PolicyFields_Privileged{
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
				Categories: []string{"Image Assurance", "Privileges Capabilities"},
				Fields: &v1.PolicyFields{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
					SetPrivileged: &v1.PolicyFields_Privileged{
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
				Categories: []string{"Image Assurance", "Privileges Capabilities"},
				Fields: &v1.PolicyFields{
					ImageName: &v1.ImageNamePolicy{
						Tag: "latest",
					},
					SetPrivileged: &v1.PolicyFields_Privileged{
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
				Categories: []string{"Image Assurance"},
				Fields: &v1.PolicyFields{
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
				Categories: []string{"Image Assurance"},
				Fields: &v1.PolicyFields{
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
				Categories: []string{"Image Assurance"},
				Fields: &v1.PolicyFields{
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
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
					{
						Key:   "foo",
						Value: "bar",
					},
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
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
					{
						Key:   "foo",
						Value: "bar",
					},
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
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
					{
						Key:   "foo",
						Value: "bar",
					},
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
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
					{
						Key:   "foo",
						Value: "bar",
					},
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
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
					{
						Key:   "foo",
						Value: "bar",
					},
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
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
					{
						Key:   "foo",
						Value: "bar",
					},
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
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
					{
						Key:   "foo",
						Value: "bar",
					},
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
				Labels: []*v1.Deployment_KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
					{
						Key:   "foo",
						Value: "bar",
					},
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
