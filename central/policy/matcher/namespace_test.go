package matcher

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestNamespaceMatcher(t *testing.T) {
	cases := []struct {
		policy    *storage.Policy
		namespace *storage.NamespaceMetadata
		matches   bool
	}{
		{
			namespace: &storage.NamespaceMetadata{
				ClusterId: "cluster1",
				Name:      "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						ClusterScope:   &storage.Scope_Cluster{Cluster: "cluster1"},
						NamespaceScope: &storage.Scope_Namespace{Namespace: "ns1"},
					},
				},
			},
			matches: true,
		},
		{
			namespace: &storage.NamespaceMetadata{
				ClusterId: "cluster1",
				Name:      "ns1",
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
			namespace: &storage.NamespaceMetadata{
				ClusterId: "cluster1",
				Name:      "ns1",
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
			namespace: &storage.NamespaceMetadata{
				ClusterId: "cluster1",
				Name:      "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						NamespaceScope: &storage.Scope_Namespace{Namespace: "ns2"},
					},
				},
			},
			matches: false,
		},
	}

	for _, c := range cases {
		actual := NewNamespaceMatcher(c.namespace).IsPolicyApplicable(c.policy)
		assert.Equal(t, c.matches, actual)
	}
}

func TestNamespaceMatcherWithWhitespace(t *testing.T) {
	cases := []struct {
		policy    *storage.Policy
		namespace *storage.NamespaceMetadata
		matches   bool
	}{
		{
			namespace: &storage.NamespaceMetadata{
				ClusterId: "cluster1",
				Name:      "ns1",
			},
			policy: &storage.Policy{
				Scope: []*storage.Scope{
					{
						ClusterScope:   &storage.Scope_Cluster{Cluster: "cluster1"},
						NamespaceScope: &storage.Scope_Namespace{Namespace: "ns1"},
					},
				},
			},
			matches: true,
		},
		{
			namespace: &storage.NamespaceMetadata{
				ClusterId: "cluster1",
				Name:      "ns1",
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
			namespace: &storage.NamespaceMetadata{
				ClusterId: "cluster1",
				Name:      "ns1",
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
							Scope: &storage.Scope{
								ClusterScope:   &storage.Scope_Cluster{Cluster: "cluster1"},
								NamespaceScope: &storage.Scope_Namespace{Namespace: "ns1"},
							},
						},
					},
				},
			},
			matches: false,
		},
		{
			namespace: &storage.NamespaceMetadata{
				ClusterId: "cluster1",
				Name:      "ns1",
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
							Scope: &storage.Scope{
								ClusterScope: &storage.Scope_Cluster{Cluster: "cluster1"},
							},
						},
					},
				},
			},
			matches: false,
		},
	}

	for _, c := range cases {
		actual := NewNamespaceMatcher(c.namespace).IsPolicyApplicable(c.policy)
		assert.Equal(t, c.matches, actual)
	}
}
