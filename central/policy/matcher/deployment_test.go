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
						ClusterScope: &storage.Scope_Cluster{Cluster: "cluster1"},
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
						ClusterScope:   &storage.Scope_Cluster{Cluster: "cluster2"},
						NamespaceScope: &storage.Scope_Namespace{Namespace: "ns1"},
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
						NamespaceScope: &storage.Scope_Namespace{Namespace: "ns1"},
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
						ClusterScope: &storage.Scope_Cluster{Cluster: "cluster1"},
					},
				},
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{
							Scope: &storage.Scope{
								NamespaceScope: &storage.Scope_Namespace{Namespace: "ns.*"},
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
						ClusterScope:   &storage.Scope_Cluster{Cluster: "cluster1"},
						NamespaceScope: &storage.Scope_Namespace{Namespace: "ns1"},
					},
				},
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{
							Name: "deployment2",
							Scope: &storage.Scope{
								NamespaceScope: &storage.Scope_Namespace{Namespace: "ns.*"},
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
						NamespaceScope: &storage.Scope_Namespace{Namespace: "ns1"},
					},
				},
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{
							Name: "deployment2",
							Scope: &storage.Scope{
								NamespaceScope: &storage.Scope_Namespace{Namespace: "ns1"},
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
						NamespaceScope: &storage.Scope_Namespace{Namespace: "ns1"},
					},
				},
				Exclusions: []*storage.Exclusion{
					{
						Deployment: &storage.Exclusion_Deployment{
							Name: "deployment2",
							Scope: &storage.Scope{
								NamespaceScope: &storage.Scope_Namespace{Namespace: "ns1"},
							},
						},
					},
					{
						Deployment: &storage.Exclusion_Deployment{
							Scope: &storage.Scope{
								NamespaceScope: &storage.Scope_Namespace{Namespace: "ns1"},
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
