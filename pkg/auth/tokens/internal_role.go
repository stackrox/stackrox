package tokens

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

// ClusterScope is the scope of a negotiated internal role on a given cluster.
// The scope can be cluster-wide (include all namespaces on the cluster),
// or restricted to a specific set of namespaces.
type ClusterScope struct {
	ClusterName       string   `json:"cluster_name"`
	ClusterFullAccess bool     `json:"cluster_full_access"`
	Namespaces        []string `json:"namespaces"`
}

// ClusterScopes is a representation of an access scope tree.
// The tree root is the map itself.
// Each cluster in the access scope is identified by its cluster ID
// The accessible namespaces within a cluster are represented
// by the map entry for the cluster ID. The map entry is a list
// of namespace names. If the list contains the wildcard value ("*"),
// then all namespaces within that cluster should be accessible.
type ClusterScopes map[string][]string

var _ permissions.ResolvedRole = (*InternalRole)(nil)

// InternalRole represents claims that materialize a negotiated ephemeral role for internal use.
type InternalRole struct {
	RoleName      string          `json:"name"`
	ClusterScopes []*ClusterScope `json:"cluster_scopes"`
	// Target token structure
	ReadResources  []string      `json:"reads,omitempty"`
	WriteResources []string      `json:"writes,omitempty"`
	Clusters       ClusterScopes `json:"clusters,omitempty"`
}

func (r *InternalRole) GetRoleName() string {
	if r == nil {
		return ""
	}
	return r.RoleName
}

func (r *InternalRole) GetPermissions() map[string]storage.Access {
	if r == nil {
		return nil
	}
	rolePermissions := make(map[string]storage.Access)
	for _, resource := range r.ReadResources {
		rolePermissions[resource] = storage.Access_READ_ACCESS
	}
	for _, resource := range r.WriteResources {
		rolePermissions[resource] = storage.Access_READ_WRITE_ACCESS
	}
	return rolePermissions
}

func (r *InternalRole) GetAccessScope() *storage.SimpleAccessScope {
	if r == nil {
		return nil
	}
	includedClusters := make([]string, 0)
	includedNamespaces := make([]*storage.SimpleAccessScope_Rules_Namespace, 0)
	for _, clusterScope := range r.ClusterScopes {
		if clusterScope == nil {
			continue
		}
		if clusterScope.ClusterName == "" {
			continue
		}
		if clusterScope.ClusterFullAccess {
			includedClusters = append(includedClusters, clusterScope.ClusterName)
		} else {
			for _, namespace := range clusterScope.Namespaces {
				scopeNamespace := &storage.SimpleAccessScope_Rules_Namespace{
					ClusterName:   clusterScope.ClusterName,
					NamespaceName: namespace,
				}
				includedNamespaces = append(includedNamespaces, scopeNamespace)
			}
		}
	}
	return &storage.SimpleAccessScope{
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters:   includedClusters,
			IncludedNamespaces: includedNamespaces,
		},
	}
}
