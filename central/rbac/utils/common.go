package utils

import (
	"context"
	"strconv"

	"github.com/stackrox/rox/central/rbac/k8srole/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

// getRolesForRoleBindings will retrieve all roles referenced in the given role bindings.
// If a namespace is given, the context used to retrieve cluster roles will be elevated, in assumption
// that the role bindings were scoped to a specific namespace and referencing a cluster role.
// If the namespace is set to empty, the given context will be used to retrieve the cluster roles.
func getRolesForRoleBindings(ctx context.Context, roleStore datastore.DataStore,
	bindings []search.Result, clusterID string, namespace string) []*storage.K8SRole {
	roleIDs, clusterRoleIDs := getRoleIDsFromBindings(bindings)
	roles := make([]*storage.K8SRole, 0, roleIDs.Cardinality()+clusterRoleIDs.Cardinality())

	// Fetch the roles without elevating the context.
	// Only attempt to fetch namespace scoped role bindings if a namespace is given.
	if namespace != "" {
		roles = append(roles, fetchRoles(ctx, roleStore, roleIDs)...)
	}

	clusterRoleCtx := ctx
	if namespace != "" {
		// For namespaced cluster role bindings, we need to potentially elevate the contexts access scope.
		// It could be that an access scope is constrained to a single namespace. If any role binding exists within that
		// namespace, we currently would be unable to list the permissions whatever is bound to that role binding has
		// (e.g. service account, user, group etc.).
		// Since we received feedback that this is an appreciated information, we will elevate the context here and give
		// READ access to the whole cluster under the following conditions:
		// - the current access scope doesn't allow READ access to K8S roles for the whole cluster.
		// - the READ access to the whole cluster will only be applicable for cluster roles.
		// - the context will not be propagated afterwards to the client.
		clusterK8SRoleScopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).
			Resource(resources.K8sRole).ClusterID(clusterID)
		if !clusterK8SRoleScopeChecker.IsAllowed() {
			clusterRoleCtx = sac.WithGlobalAccessScopeChecker(ctx, sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.K8sRole), sac.ClusterScopeKeys(clusterID)))
		}
	}

	// Fetch cluster roles with potentially elevated context.
	roles = append(roles, fetchRoles(clusterRoleCtx, roleStore, clusterRoleIDs)...)

	return roles
}

func getRoleIDsFromBindings(bindingResults []search.Result) (set.StringSet, set.StringSet) {
	roleIDs := set.NewStringSet()
	clusterRoleIDs := set.NewStringSet()
	for _, bindingResult := range bindingResults {
		roleID, clusterRole := convertRoleBindingSearchResult(bindingResult)
		if roleID != "" {
			// In cae of evaluating namespace permission where a specific namespace will be set, we will filter relevant
			// bindings to cluster roles to the specific namespace.
			// In case of evaluating cluster permission where no namespace will be set, we will only look at bindings
			// without a namespace being set, since bindings bound to a specific namespace are not relevant.
			if clusterRole {
				clusterRoleIDs.Add(roleID)
			} else {
				roleIDs.Add(roleID)
			}
		}
	}
	return roleIDs, clusterRoleIDs
}

func fetchRoles(ctx context.Context, roleStore datastore.DataStore, roleIDs set.StringSet) []*storage.K8SRole {
	roles := make([]*storage.K8SRole, 0, roleIDs.Cardinality())
	for roleID := range roleIDs {
		role, exists, err := roleStore.GetRole(ctx, roleID)
		if exists && err == nil {
			roles = append(roles, role)
		}
	}
	return roles
}

func convertRoleBindingSearchResult(r search.Result) (string, bool) {
	var (
		roleID      string
		clusterRole bool
	)
	roleIDMatches := r.Matches[schema.RoleBindingsSchema.OptionsMap.MustGet(search.RoleID.String()).GetFieldPath()]
	if len(roleIDMatches) == 1 {
		roleID = roleIDMatches[0]
	}
	clusterRoleMatches := r.Matches[schema.RoleBindingsSchema.OptionsMap.
		MustGet(search.ClusterRole.String()).GetFieldPath()]
	if len(clusterRoleMatches) == 1 {
		clusterRole, _ = strconv.ParseBool(clusterRoleMatches[0])
	}
	return roleID, clusterRole
}
