package authorizer

import (
	"context"
	"testing"

	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	rolePkg "github.com/stackrox/rox/pkg/auth/role"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/observe"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
)

var (
	firstCluster = payload.Cluster{
		ID:   "cluster-1",
		Name: "FirstCluster",
	}
	secondCluster = payload.Cluster{
		ID:   "cluster-2",
		Name: "SecondCluster",
	}

	firstNamespaceName  = "FirstNamespace"
	secondNamespaceName = "SecondNamespace"

	allResourcesView = mapResourcesToAccess(resources.AllResourcesViewPermissions())

	clusters = []*storage.Cluster{
		{Id: firstCluster.ID, Name: firstCluster.Name},
		{Id: secondCluster.ID, Name: secondCluster.Name},
	}
	namespaces = []*storage.NamespaceMetadata{{
		Id:          "namespace-1",
		Name:        firstNamespaceName,
		ClusterId:   firstCluster.ID,
		ClusterName: firstCluster.Name,
	}, {
		Id:          "namespace-2",
		Name:        secondNamespaceName,
		ClusterId:   firstCluster.ID,
		ClusterName: firstCluster.Name,
	}}
)

func TestBuiltInScopeAuthorizerWithTracing(t *testing.T) {
	t.Parallel()
	clusterEdit := map[string]storage.Access{string(resources.Cluster.Resource): storage.Access_READ_WRITE_ACCESS}

	tests := []struct {
		name      string
		roles     []permissions.ResolvedRole
		scopeKeys []sac.ScopeKey
		results   []sac.TryAllowedResult
	}{
		{
			name:      "allow read from cluster with permissions",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Cluster())},
			scopeKeys: readCluster(firstCluster.ID),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Allow},
		},
		{
			name:      "allow cluster modification (e.g., creation) with permissions even if it does not exist yet",
			roles:     []permissions.ResolvedRole{role(clusterEdit, rolePkg.AccessScopeIncludeAll)},
			scopeKeys: scopeKeys(storage.Access_READ_WRITE_ACCESS, resources.Cluster.Resource, "unknown ID", ""),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Allow, sac.Allow, sac.Allow},
		},
		{
			name:      "deny cluster view with permissions but no access scope if id does not exist",
			roles:     []permissions.ResolvedRole{role(clusterEdit, withAccessTo1Cluster())},
			scopeKeys: readCluster("unknown ID"),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny},
		},
		{
			name:      "deny cluster modification with permission to view",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Cluster())},
			scopeKeys: scopeKeys(storage.Access_READ_WRITE_ACCESS, resources.Cluster.Resource, firstCluster.ID, ""),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny, sac.Deny},
		},
		{
			name:      "deny read from cluster with no scope access",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Cluster())},
			scopeKeys: readCluster(secondCluster.ID),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny},
		},
		{
			name: "allow read from namespace with multiple roles",
			roles: []permissions.ResolvedRole{
				role(allResourcesView, withAccessTo1Namespace()),
				role(allResourcesView, withAccessTo1Cluster())},
			scopeKeys: readNamespace(firstCluster.ID, secondNamespaceName),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Allow, sac.Allow},
		},
		{
			name:      "allow read from anything when scope unrestricted",
			roles:     []permissions.ResolvedRole{role(allResourcesView, rolePkg.AccessScopeIncludeAll)},
			scopeKeys: readCluster("unknown ID"),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Allow, sac.Allow},
		},
		{
			name:      "deny read from anything when scope is nil",
			roles:     []permissions.ResolvedRole{role(allResourcesView, nil)},
			scopeKeys: readCluster("unknown ID"),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny},
		},
		{
			name:      "deny read from anything when scope is empty",
			roles:     []permissions.ResolvedRole{role(allResourcesView, &storage.SimpleAccessScope{Id: "empty"})},
			scopeKeys: readCluster(firstCluster.ID),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny},
		},
		{
			name:      "deny read from anything when scope deny all",
			roles:     []permissions.ResolvedRole{role(allResourcesView, rolePkg.AccessScopeExcludeAll)},
			scopeKeys: readCluster(firstCluster.ID),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			trace := observe.NewAuthzTrace()
			scc := newGlobalScopeCheckerCore(clusters, namespaces, tc.roles, trace)
			for i, scopeKey := range tc.scopeKeys {
				scc = scc.SubScopeChecker(scopeKey)
				expected := tc.results[i]
				got := scc.TryAllowed()
				assert.Equalf(t, expected, got, "expected %d, got %d for scope %s, level [%d]", expected, got, scopeKey, i)
				err := scc.PerformChecks(context.Background())
				wantErr := expected == sac.Unknown
				assert.Truef(t, (err != nil) == wantErr, "got %+v", err)
			}
			// The amount of "allowed" traces should equal the amount of
			// expected Allow responses.
			assert.Equal(t, countAllowedResults(tc.results), observe.CountAllowedTraces(trace), "number of allowed traces differs from the number of expected allow responses")
		})
	}
}

func TestScopeCheckerWithParallelAccessAndSharedGlobalScopeChecker(t *testing.T) {
	t.Parallel()
	roles := []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Namespace())}

	subScopeChecker := newGlobalScopeCheckerCore(clusters, namespaces, roles, nil)

	tests := []struct {
		name      string
		scopeKeys []sac.ScopeKey
		results   []sac.TryAllowedResult
	}{
		{
			name:      "allow read from cluster with partial access",
			scopeKeys: readCluster(firstCluster.ID),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Allow},
		},
		{
			name:      "allow read from namespace with direct access",
			scopeKeys: readNamespace(firstCluster.ID, firstNamespaceName),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny, sac.Allow},
		},
		{
			name: "deny read from global",
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			},
			results: []sac.TryAllowedResult{sac.Deny},
		},
		{
			name:      "error when wrong sub scope",
			scopeKeys: sac.ClusterScopeKeys(firstCluster.ID),
			results:   []sac.TryAllowedResult{sac.Unknown},
		},
		{
			name: "error when wrong sub scope",
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey("unknown resource"),
			},
			results: []sac.TryAllowedResult{sac.Deny, sac.Unknown},
		},
		{
			name: "error when wrong sub scope",
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ClusterScopeKey(firstCluster.ID),
			},
			results: []sac.TryAllowedResult{sac.Deny, sac.Unknown},
		},
		{
			name: "error when wrong sub scope",
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resources.Cluster.Resource),
				sac.NamespaceScopeKey(firstNamespaceName),
			},
			results: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Unknown},
		},
		{
			name: "error when wrong sub scope",
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resources.Cluster.Resource),
				sac.ClusterScopeKey(firstCluster.ID),
				sac.ClusterScopeKey(secondCluster.ID),
			},
			results: []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Allow, sac.Unknown},
		},
		{
			name:      "deny when unknown namespace",
			scopeKeys: readNamespace(firstCluster.ID, "unknown ID"),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny, sac.Deny},
		},
		{
			name:      "deny when empty namespace",
			scopeKeys: readNamespace(firstCluster.ID, ""),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Deny, sac.Deny, sac.Deny},
		},
		{
			name: "allow when global scope resource",
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resources.APIToken.Resource),
			},
			results: []sac.TryAllowedResult{sac.Deny, sac.Allow},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scc := subScopeChecker
			assert.Len(t, tc.results, len(tc.scopeKeys), "results and scope keys must have same length")
			for i, scopeKey := range tc.scopeKeys {
				scc = scc.SubScopeChecker(scopeKey)
				expected := tc.results[i]
				assert.Equalf(t, expected, scc.TryAllowed(), "scope %s, level [%d]", scopeKey, i)
				err := scc.PerformChecks(context.Background())
				wantErr := expected == sac.Unknown
				assert.Truef(t, (err != nil) == wantErr, "got %+v", err)
			}
		})
	}
}

func TestGlobalScopeCheckerCore(t *testing.T) {
	t.Parallel()
	scc := newGlobalScopeCheckerCore(nil, nil, nil, nil)
	assert.Equal(t, nil, scc.PerformChecks(context.Background()))
	assert.Equal(t, sac.Deny, scc.TryAllowed())
}

func TestBuiltInScopeAuthorizerPanicsWhenErrorOnComputeAccessScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		roles     []permissions.ResolvedRole
		scopeKeys []sac.ScopeKey
		results   []sac.TryAllowedResult
	}{
		{
			name: "error when could not compute effective access scope",
			roles: []permissions.ResolvedRole{role(allResourcesView, &storage.SimpleAccessScope{
				Id: "with-invalid-key",
				Rules: &storage.SimpleAccessScope_Rules{
					ClusterLabelSelectors: []*storage.SetBasedLabelSelector{{
						Requirements: []*storage.SetBasedLabelSelector_Requirement{
							{Key: "invalid key"},
						}}}}})},
			scopeKeys: readCluster(firstCluster.ID),
			results:   []sac.TryAllowedResult{sac.Deny, sac.Unknown, sac.Unknown, sac.Unknown},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scc := newGlobalScopeCheckerCore(clusters, namespaces, tc.roles, nil)
			for i, scopeKey := range tc.scopeKeys {
				scc = scc.SubScopeChecker(scopeKey)
				expected := tc.results[i]
				if expected == sac.Unknown && !buildinfo.ReleaseBuild {
					assert.Panics(t, func() { scc.TryAllowed() })
				} else {
					assert.Equal(t, expected, scc.TryAllowed())
				}
				err := scc.PerformChecks(context.Background())
				assert.NoError(t, err)
			}
		})
	}
}

func readCluster(clusterID string) []sac.ScopeKey {
	return scopeKeys(storage.Access_READ_ACCESS, resources.Cluster.Resource, clusterID, "")[:3]
}

func readNamespace(clusterID, namespaceName string) []sac.ScopeKey {
	return scopeKeys(storage.Access_READ_ACCESS, resources.Namespace.Resource, clusterID, namespaceName)
}

func scopeKeys(access storage.Access, res permissions.Resource, clusterID, namespaceName string) []sac.ScopeKey {
	return []sac.ScopeKey{
		sac.AccessModeScopeKey(access),
		sac.ResourceScopeKey(res),
		sac.ClusterScopeKey(clusterID),
		sac.NamespaceScopeKey(namespaceName),
	}
}

func withAccessTo1Cluster() *storage.SimpleAccessScope {
	return &storage.SimpleAccessScope{
		Id: "withAccessTo1Cluster",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{firstCluster.Name},
		},
	}
}

func withAccessTo1Namespace() *storage.SimpleAccessScope {
	return &storage.SimpleAccessScope{
		Id: "withAccessTo1Namespace",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{
				ClusterName:   firstCluster.Name,
				NamespaceName: firstNamespaceName,
			}},
		},
	}
}

func role(perms map[string]storage.Access, as *storage.SimpleAccessScope) permissions.ResolvedRole {
	return roleWithName("test-role", perms, as)
}

func roleWithName(name string, perms map[string]storage.Access, as *storage.SimpleAccessScope) permissions.ResolvedRole {
	return roletest.NewResolvedRole(name, perms, as)
}

func mapResourcesToAccess(res []permissions.ResourceWithAccess) map[string]storage.Access {
	idToAccess := make(map[string]storage.Access, len(res))
	for _, rwa := range res {
		idToAccess[rwa.Resource.String()] = rwa.Access
	}
	return idToAccess
}

func countAllowedResults(xs []sac.TryAllowedResult) int {
	result := 0
	for _, x := range xs {
		if x == sac.Allow {
			result++
		}
	}
	return result
}
