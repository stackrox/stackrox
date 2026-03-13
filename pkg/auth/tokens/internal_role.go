package tokens

import (
	"encoding/json"
	"slices"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/set"
)

// ClusterScopes is a representation of an access scope tree.
// The tree root is the map itself.
// Each cluster in the access scope is identified by its cluster ID
// The accessible namespaces within a cluster are represented
// by the map entry for the cluster ID. The map entry is a list
// of namespace names. If the list contains the wildcard value ("*"),
// then all namespaces within that cluster should be accessible.
type ClusterScopes map[string][]string

var _ permissions.ResolvedRole = (*InternalRole)(nil)

type AccessWrapper storage.Access

func (a *AccessWrapper) MarshalJSON() ([]byte, error) {
	if a == nil {
		return json.Marshal("null")
	}
	access := storage.Access(*a)
	switch access {
	case storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS:
		return json.Marshal(access.String())
	default:
		return json.Marshal(storage.Access_NO_ACCESS.String())
	}
}

func (a *AccessWrapper) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	access := storage.Access_NO_ACCESS
	if asInt, found := storage.Access_value[s]; found {
		access = storage.Access(asInt)
	}
	*a = AccessWrapper(access)
	return nil
}

func (a *AccessWrapper) AsAccess() storage.Access {
	if a == nil {
		return storage.Access_NO_ACCESS
	}
	switch storage.Access(*a) {
	case storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS:
		return storage.Access(*a)
	default:
		return storage.Access_NO_ACCESS
	}
}

// InternalRole represents claims that materialize a negotiated ephemeral role for internal use.
type InternalRole struct {
	RoleName    string                     `json:"name,omitempty"`
	Permissions map[AccessWrapper][]string `json:"permissions,omitempty"`
	// The key for this cluster scope map is the cluster ID.
	// TODO: Uncomment when access scope selection rules allow Cluster ID.
	// Clusters ClusterScopes `json:"clusters,omitempty"`
	// The key for this cluster scope map is the cluster Name.
	ClustersByName ClusterScopes `json:"named_clusters,omitempty"`
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
	permissionCount := 0
	for _, targetResources := range r.Permissions {
		permissionCount += len(targetResources)
	}
	rolePermissions := make(map[string]storage.Access, permissionCount)
	for level, resources := range r.Permissions {
		accessLevel := level.AsAccess()
		for _, resource := range resources {
			prevLevel := rolePermissions[resource]
			if prevLevel <= accessLevel {
				rolePermissions[resource] = accessLevel
			}
		}
	}
	return rolePermissions
}

func (r *InternalRole) GetAccessScope() *storage.SimpleAccessScope {
	if r == nil {
		return nil
	}
	includedClusterNames := set.NewStringSet()
	includedNamespacesByClusterNames := make(map[string]set.StringSet)
	for clusterName, namespaces := range r.ClustersByName {
		fullAccess := false
		for _, ns := range namespaces {
			if ns == "*" {
				fullAccess = true
				break
			}
		}
		if fullAccess {
			includedClusterNames.Add(clusterName)
			continue
		}
		clusterNamespaces := set.NewStringSet(namespaces...)
		includedNamespacesByClusterNames[clusterName] = clusterNamespaces
	}

	includedNamespaces := make([]*storage.SimpleAccessScope_Rules_Namespace, 0, len(includedNamespacesByClusterNames))
	sortedPartialClusterNames := make([]string, 0, len(includedNamespacesByClusterNames))
	for clusterName := range includedNamespacesByClusterNames {
		sortedPartialClusterNames = append(sortedPartialClusterNames, clusterName)
	}
	slices.Sort(sortedPartialClusterNames)
	stringSort := func(i, j string) bool { return i < j }
	for _, clusterName := range sortedPartialClusterNames {
		for _, ns := range includedNamespacesByClusterNames[clusterName].AsSortedSlice(stringSort) {
			includedNamespaces = append(includedNamespaces, &storage.SimpleAccessScope_Rules_Namespace{
				ClusterName:   clusterName,
				NamespaceName: ns,
			})
		}
	}

	return &storage.SimpleAccessScope{
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters:   includedClusterNames.AsSortedSlice(stringSort),
			IncludedNamespaces: includedNamespaces,
		},
	}
}
