package externalrolebroker

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	clusterviewv1alpha1 "github.com/stolostron/cluster-lifecycle-api/clusterview/v1alpha1"
)

// ConvertBindingsToSimpleAccessScope converts ClusterBinding entries from an ACM UserPermission
// to a storage.SimpleAccessScope.
//
// The function creates:
//   - Entries in IncludedClusters for bindings with cluster scope
//   - Entries in IncludedNamespaces for bindings with namespace scope
//
// Note: In ACM, the Cluster field in a ClusterBinding contains the managed cluster name,
// which corresponds to a namespace name on the hub cluster. This is mapped to ClusterName
// in the SimpleAccessScope structure.
//
// The returned SimpleAccessScope has a generated ID and empty name/description.
func ConvertBindingsToSimpleAccessScope(bindings []clusterviewv1alpha1.ClusterBinding) *storage.SimpleAccessScope {
	// Use sets to avoid duplicates
	clusterSet := set.NewStringSet()
	var namespaces []*storage.SimpleAccessScope_Rules_Namespace

	for _, binding := range bindings {
		switch binding.Scope {
		case clusterviewv1alpha1.BindingScopeCluster:
			// Cluster-scoped binding: add to IncludedClusters
			clusterSet.Add(binding.Cluster)

		case clusterviewv1alpha1.BindingScopeNamespace:
			// Namespace-scoped binding: create entries in IncludedNamespaces
			for _, ns := range binding.Namespaces {
				// Skip wildcard namespaces - they should be cluster-scoped instead
				if ns == "*" {
					continue
				}

				namespaces = append(namespaces, &storage.SimpleAccessScope_Rules_Namespace{
					ClusterName:   binding.Cluster,
					NamespaceName: ns,
				})
			}
		}
	}

	return &storage.SimpleAccessScope{
		Id:          uuid.NewV4().String(),
		Name:        "",
		Description: "",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters:   clusterSet.AsSlice(),
			IncludedNamespaces: namespaces,
		},
	}
}
