package matcher

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentMatcher(t *testing.T) {
	cases := []struct {
		policy     *storage.Policy
		deployment *storage.Deployment
		matches    bool
	}{
		{
			deployment: storage.Deployment_builder{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			}.Build(),
			policy: storage.Policy_builder{
				Scope: []*storage.Scope{
					storage.Scope_builder{
						Cluster: "cluster1",
					}.Build(),
				},
			}.Build(),
			matches: true,
		},
		{
			deployment: storage.Deployment_builder{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			}.Build(),
			policy: storage.Policy_builder{
				Scope: []*storage.Scope{
					storage.Scope_builder{
						Cluster:   "cluster2",
						Namespace: "ns1",
					}.Build(),
				},
			}.Build(),
			matches: false,
		},
		{
			deployment: storage.Deployment_builder{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			}.Build(),
			policy: storage.Policy_builder{
				Scope: []*storage.Scope{
					storage.Scope_builder{
						Namespace: "ns1",
					}.Build(),
				},
			}.Build(),
			matches: true,
		},
		{
			deployment: storage.Deployment_builder{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			}.Build(),
			policy:  &storage.Policy{},
			matches: true,
		},
	}

	for _, c := range cases {
		actual := NewDeploymentMatcher(c.deployment).IsPolicyApplicable(c.policy)
		assert.Equal(t, c.matches, actual)
	}
}

func TestDeploymentWithExclusion(t *testing.T) {
	cases := []struct {
		policy     *storage.Policy
		deployment *storage.Deployment
		matches    bool
	}{
		{
			deployment: storage.Deployment_builder{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			}.Build(),
			policy: storage.Policy_builder{
				Scope: []*storage.Scope{
					storage.Scope_builder{
						Cluster: "cluster1",
					}.Build(),
				},
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{
							Scope: storage.Scope_builder{
								Namespace: "ns.*",
							}.Build(),
						}.Build(),
					}.Build(),
				},
			}.Build(),
			matches: false,
		},
		{
			deployment: storage.Deployment_builder{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			}.Build(),
			policy: storage.Policy_builder{
				Scope: []*storage.Scope{
					storage.Scope_builder{
						Cluster:   "cluster1",
						Namespace: "ns1",
					}.Build(),
				},
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{
							Name: "deployment2",
							Scope: storage.Scope_builder{
								Namespace: "ns.*",
							}.Build(),
						}.Build(),
					}.Build(),
				},
			}.Build(),
			matches: true,
		},
		{
			deployment: storage.Deployment_builder{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			}.Build(),
			policy: storage.Policy_builder{
				Scope: []*storage.Scope{
					storage.Scope_builder{
						Namespace: "ns1",
					}.Build(),
				},
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{
							Name: "deployment2",
							Scope: storage.Scope_builder{
								Namespace: "ns1",
							}.Build(),
						}.Build(),
					}.Build(),
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{
							Name: "deployment1",
						}.Build(),
					}.Build(),
				},
			}.Build(),
			matches: false,
		},
		{
			deployment: storage.Deployment_builder{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			}.Build(),
			policy: storage.Policy_builder{
				Scope: []*storage.Scope{
					storage.Scope_builder{
						Namespace: "ns1",
					}.Build(),
				},
				Exclusions: []*storage.Exclusion{
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{
							Name: "deployment2",
							Scope: storage.Scope_builder{
								Namespace: "ns1",
							}.Build(),
						}.Build(),
					}.Build(),
					storage.Exclusion_builder{
						Deployment: storage.Exclusion_Deployment_builder{
							Scope: storage.Scope_builder{
								Namespace: "ns1",
							}.Build(),
						}.Build(),
					}.Build(),
				},
			}.Build(),
			matches: false,
		},
		{
			deployment: storage.Deployment_builder{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			}.Build(),
			policy:  &storage.Policy{},
			matches: true,
		},
	}

	for _, c := range cases {
		actual := NewDeploymentMatcher(c.deployment).IsPolicyApplicable(c.policy)
		assert.Equal(t, c.matches, actual)
	}
}
