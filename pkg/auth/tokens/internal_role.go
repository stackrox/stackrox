package tokens

import (
	"encoding/json"
	"slices"
	"strings"

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

// InternalRole represents claims that materialize a negotiated ephemeral role for internal use.
type InternalRole struct {
	RoleName    string                      `json:"name,omitempty"`
	Permissions map[storage.Access][]string `json:"permissions,omitempty"`
	// The key for this cluster scope map is the cluster ID.
	// TODO: Uncomment when access scope selection rules allow Cluster ID.
	// Clusters ClusterScopes `json:"clusters,omitempty"`
	// The key for this cluster scope map is the cluster Name.
	ClustersByName ClusterScopes `json:"named_clusters,omitempty"`
}

// MarshalJSON implements json.Marshaler for InternalRole.
// This is necessary because go-jose doesn't support map[storage.Access][]string directly.
func (r *InternalRole) MarshalJSON() ([]byte, error) {
	// Create a temporary struct with string-keyed permissions map
	type Alias InternalRole
	aux := &struct {
		Permissions map[string][]string `json:"permissions,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	// Convert map[storage.Access][]string to map[string][]string
	// Validate and normalize access levels during marshaling
	if r.Permissions != nil {
		aux.Permissions = make(map[string][]string, len(r.Permissions))
		for k, v := range r.Permissions {
			validatedAccess := validateAccessLevel(k)
			key := "none"
			if validatedAccess != storage.Access_NO_ACCESS {
				key = validatedAccess.String()
				key = strings.TrimSuffix(key, "_ACCESS")
				key = strings.ToLower(key)
				key = strings.ReplaceAll(key, "_", "-")
			}
			aux.Permissions[key] = v
		}
	}

	return json.Marshal(aux)
}

// UnmarshalJSON implements json.Unmarshaler for InternalRole.
func (r *InternalRole) UnmarshalJSON(data []byte) error {
	// Create a temporary struct with string-keyed permissions map
	type Alias InternalRole
	aux := &struct {
		Permissions map[string][]string `json:"permissions,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Convert map[string][]string to map[storage.Access][]string
	if aux.Permissions != nil {
		r.Permissions = make(map[storage.Access][]string, len(aux.Permissions))
		for k, v := range aux.Permissions {
			// Parse the access level from string
			access := storage.Access_NO_ACCESS
			key := k
			key = strings.ReplaceAll(key, "-", "_")
			key = strings.ToUpper(key)
			key += "_ACCESS"
			if asInt, found := storage.Access_value[key]; found {
				access = storage.Access(asInt)
			}
			r.Permissions[access] = v
		}
	}

	return nil
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
		// Validate that the access level is a known enum value, default to NO_ACCESS
		accessLevel := validateAccessLevel(level)
		for _, resource := range resources {
			prevLevel := rolePermissions[resource]
			if prevLevel <= accessLevel {
				rolePermissions[resource] = accessLevel
			}
		}
	}
	return rolePermissions
}

// validateAccessLevel ensures the access level is a valid storage.Access enum value.
// Returns the access level if valid (READ_ACCESS or READ_WRITE_ACCESS), otherwise NO_ACCESS.
func validateAccessLevel(access storage.Access) storage.Access {
	switch access {
	case storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS:
		return access
	default:
		return storage.Access_NO_ACCESS
	}
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
