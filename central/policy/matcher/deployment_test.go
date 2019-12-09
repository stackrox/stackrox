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

func TestDeploymentWithWhitelist(t *testing.T) {
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
				Whitelists: []*storage.Whitelist{
					{
						Deployment: &storage.Whitelist_Deployment{
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
				Whitelists: []*storage.Whitelist{
					{
						Deployment: &storage.Whitelist_Deployment{
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
				Whitelists: []*storage.Whitelist{
					{
						Deployment: &storage.Whitelist_Deployment{
							Name: "deployment2",
							Scope: &storage.Scope{
								Namespace: "ns1",
							},
						},
					},
					{
						Deployment: &storage.Whitelist_Deployment{
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
				Whitelists: []*storage.Whitelist{
					{
						Deployment: &storage.Whitelist_Deployment{
							Name: "deployment2",
							Scope: &storage.Scope{
								Namespace: "ns1",
							},
						},
					},
					{
						Deployment: &storage.Whitelist_Deployment{
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
