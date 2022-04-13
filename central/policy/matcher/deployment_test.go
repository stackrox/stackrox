package matcher

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentMatcher(t *testing.T) {
	cases := []struct {
		policy     *storage.Policy
		deployment *storage.Deployment
		matches    bool
	}{
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Cluster: "cluster1",
					},
				},
			},
			matches: true,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Cluster:   "cluster2",
						Namespace: "ns1",
					},
				},
			},
			matches: false,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Namespace: "ns1",
					},
				},
			},
			matches: true,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
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
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Cluster: "cluster1",
					},
				},
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{
							Scope: &storage.Scope{
								Namespace: "ns.*",
							},
						},
					},
				},
			},
			matches: false,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Cluster:   "cluster1",
						Namespace: "ns1",
					},
				},
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{
							Name: "deployment2",
							Scope: &storage.Scope{
								Namespace: "ns.*",
							},
						},
					},
				},
			},
			matches: true,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Namespace: "ns1",
					},
				},
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{
							Name: "deployment2",
							Scope: &storage.Scope{
								Namespace: "ns1",
							},
						},
					},
					{
						Deployment: &storage.Exclusion_Deployment{
							Name: "deployment1",
						},
					},
				},
			},
			matches: false,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						Namespace: "ns1",
					},
				},
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{
							Name: "deployment2",
							Scope: &storage.Scope{
								Namespace: "ns1",
							},
						},
					},
					{
						Deployment: &storage.Exclusion_Deployment{
							Scope: &storage.Scope{
								Namespace: "ns1",
							},
						},
					},
				},
			},
			matches: false,
		},
		{
			deployment: &storage.Deployment{
				Name:      "deployment1",
				ClusterId: "cluster1",
				Namespace: "ns1",
			},
			policy:  &storage.Policy{},
			matches: true,
		},
	}

	for _, c := range cases {
		actual := NewDeploymentMatcher(c.deployment).IsPolicyApplicable(c.policy)
		assert.Equal(t, c.matches, actual)
	}
}
