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
			namespace: storage.NamespaceMetadata_builder{
				ClusterId: "cluster1",
				Name:      "ns1",
			}.Build(),
			policy: storage.Policy_builder{
				Scope: []*storage.Scope{
					storage.Scope_builder{
						Cluster:   "cluster1",
						Namespace: "ns1",
					}.Build(),
				},
			}.Build(),
			matches: true,
		},
		{
			namespace: storage.NamespaceMetadata_builder{
				ClusterId: "cluster1",
				Name:      "ns1",
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
			namespace: storage.NamespaceMetadata_builder{
				ClusterId: "cluster1",
				Name:      "ns1",
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
			namespace: storage.NamespaceMetadata_builder{
				ClusterId: "cluster1",
				Name:      "ns1",
			}.Build(),
			policy: storage.Policy_builder{
				Scope: []*storage.Scope{
					storage.Scope_builder{
						Namespace: "ns2",
					}.Build(),
				},
			}.Build(),
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
			namespace: storage.NamespaceMetadata_builder{
				ClusterId: "cluster1",
				Name:      "ns1",
			}.Build(),
			policy: storage.Policy_builder{
				Scope: []*storage.Scope{
					storage.Scope_builder{
						Cluster:   "cluster1",
						Namespace: "ns1",
					}.Build(),
				},
			}.Build(),
			matches: true,
		},
		{
			namespace: storage.NamespaceMetadata_builder{
				ClusterId: "cluster1",
				Name:      "ns1",
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
			namespace: storage.NamespaceMetadata_builder{
				ClusterId: "cluster1",
				Name:      "ns1",
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
							Scope: storage.Scope_builder{
								Cluster:   "cluster1",
								Namespace: "ns1",
							}.Build(),
						}.Build(),
					}.Build(),
				},
			}.Build(),
			matches: false,
		},
		{
			namespace: storage.NamespaceMetadata_builder{
				ClusterId: "cluster1",
				Name:      "ns1",
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
							Scope: storage.Scope_builder{
								Cluster: "cluster1",
							}.Build(),
						}.Build(),
					}.Build(),
				},
			}.Build(),
			matches: false,
		},
	}

	for _, c := range cases {
		actual := NewNamespaceMatcher(c.namespace).IsPolicyApplicable(c.policy)
		assert.Equal(t, c.matches, actual)
	}
}
