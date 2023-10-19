package authorizer

import (
	"testing"

	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/observe"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
)

const (
	firstClusterID    = "cluster-1"
	firstClusterName  = "FirstCluster"
	secondClusterID   = "cluster-2"
	secondClusterName = "SecondCluster"

	firstNamespaceName  = "FirstNamespace"
	secondNamespaceName = "SecondNamespace"
)

var (
	allResourcesView = mapResourcesToAccess(resources.AllResourcesViewPermissions())
	allResourcesEdit = mapResourcesToAccess(resources.AllResourcesModifyPermissions())

	clusters = []effectiveaccessscope.ClusterForSAC{
		&clusterForSAC{ID: firstClusterID, Name: firstClusterName},
		&clusterForSAC{ID: secondClusterID, Name: secondClusterName},
	}
	namespaces = []effectiveaccessscope.NamespaceForSAC{
		&namespaceForSAC{
			ID:          "namespace-1",
			Name:        firstNamespaceName,
			ClusterID:   firstClusterID,
			ClusterName: firstClusterName,
		},
		&namespaceForSAC{
			ID:          "namespace-2",
			Name:        secondNamespaceName,
			ClusterID:   firstClusterID,
			ClusterName: firstClusterName,
		},
	}
)

func TestBuiltInScopeAuthorizerWithTracing(t *testing.T) {
	t.Parallel()
	clusterEdit := map[string]storage.Access{string(resources.Cluster.Resource): storage.Access_READ_WRITE_ACCESS}
	complianceEdit := map[string]storage.Access{string(resources.Compliance.Resource): storage.Access_READ_WRITE_ACCESS}

	tests := []struct {
		name      string
		roles     []permissions.ResolvedRole
		scopeKeys []sac.ScopeKey
		results   []bool
	}{
		{
			name:      "allow read from cluster with permissions",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Cluster())},
			scopeKeys: readCluster(firstClusterID, resources.Cluster.Resource),
			results:   []bool{false, false, true},
		},
		{
			name:      "allow cluster modification (e.g., creation) with permissions even if it does not exist yet",
			roles:     []permissions.ResolvedRole{role(clusterEdit, rolePkg.AccessScopeIncludeAll)},
			scopeKeys: scopeKeys(storage.Access_READ_WRITE_ACCESS, resources.Cluster.Resource, "unknown ID", ""),
			results:   []bool{false, true, true, true},
		},
		{
			name:      "deny cluster view with permissions but no access scope if id does not exist",
			roles:     []permissions.ResolvedRole{role(clusterEdit, withAccessTo1Cluster())},
			scopeKeys: readCluster("unknown ID", resources.Cluster.Resource),
			results:   []bool{false, false, false},
		},
		{
			name:      "deny cluster modification with permission to view",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Cluster())},
			scopeKeys: scopeKeys(storage.Access_READ_WRITE_ACCESS, resources.Cluster.Resource, firstClusterID, ""),
			results:   []bool{false, false, false, false},
		},
		{
			name:      "deny read from cluster with no scope access",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Cluster())},
			scopeKeys: readCluster(secondClusterID, resources.Cluster.Resource),
			results:   []bool{false, false, false},
		},
		{
			name:      "allow read from compliance with permissions",
			roles:     []permissions.ResolvedRole{role(complianceEdit, withAccessTo1Cluster())},
			scopeKeys: readCluster(firstClusterID, resources.Compliance.Resource),
			results:   []bool{false, false, true},
		},
		{
			name: "allow read from namespace with multiple roles",
			roles: []permissions.ResolvedRole{
				role(allResourcesView, withAccessTo1Namespace()),
				role(allResourcesView, withAccessTo1Cluster())},
			scopeKeys: readNamespace(firstClusterID, secondNamespaceName),
			results:   []bool{false, false, true, true},
		},
		{
			name:      "allow read from anything when scope unrestricted",
			roles:     []permissions.ResolvedRole{role(allResourcesView, rolePkg.AccessScopeIncludeAll)},
			scopeKeys: readCluster("unknown ID", resources.Cluster.Resource),
			results:   []bool{false, true, true},
		},
		{
			name:      "deny read from anything when scope is nil",
			roles:     []permissions.ResolvedRole{role(allResourcesView, nil)},
			scopeKeys: readCluster("unknown ID", resources.Cluster.Resource),
			results:   []bool{false, false, false},
		},
		{
			name:      "deny read from anything when scope is empty",
			roles:     []permissions.ResolvedRole{role(allResourcesView, &storage.SimpleAccessScope{Id: "empty"})},
			scopeKeys: readCluster(firstClusterID, resources.Cluster.Resource),
			results:   []bool{false, false, false},
		},
		{
			name:      "deny read from anything when scope deny all",
			roles:     []permissions.ResolvedRole{role(allResourcesView, rolePkg.AccessScopeExcludeAll)},
			scopeKeys: readCluster(firstClusterID, resources.Cluster.Resource),
			results:   []bool{false, false, false},
		},
		{
			name:      "deny read from anything when scope deny all",
			roles:     []permissions.ResolvedRole{role(allResourcesView, rolePkg.AccessScopeExcludeAll)},
			scopeKeys: []sac.ScopeKey{sac.AccessModeScopeKey(storage.Access_READ_ACCESS), sac.ResourceScopeKey(resources.InstallationInfo.Resource)},
			results:   []bool{false, false},
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
				got := scc.Allowed()
				assert.Equalf(t, expected, got, "expected %d, got %d for scope %s, level [%d]", expected, got, scopeKey, i)
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
		results   []bool
	}{
		{
			name:      "allow read from cluster with partial access",
			scopeKeys: readCluster(firstClusterID, resources.Cluster.Resource),
			results:   []bool{false, false, true},
		},
		{
			name:      "allow read from namespace with direct access",
			scopeKeys: readNamespace(firstClusterID, firstNamespaceName),
			results:   []bool{false, false, false, true},
		},
		{
			name: "deny read from global",
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
			},
			results: []bool{false},
		},
		{
			name: "error when wrong sub scope",
			scopeKeys: []sac.ScopeKey{
				sac.ClusterScopeKey(firstClusterID),
			},
			results: []bool{},
		},
		{
			name: "error when wrong sub scope",
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey("unknown resource"),
			},
			results: []bool{false},
		},
		{
			name: "error when wrong sub scope",
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ClusterScopeKey(firstClusterID),
			},
			results: []bool{false},
		},
		{
			name: "error when wrong sub scope",
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resources.Cluster.Resource),
				sac.NamespaceScopeKey(firstNamespaceName),
			},
			results: []bool{false, false},
		},
		{
			name: "error when wrong sub scope",
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resources.Cluster.Resource),
				sac.ClusterScopeKey(firstClusterID),
				sac.ClusterScopeKey(secondClusterID),
			},
			results: []bool{false, false, true},
		},
		{
			name:      "deny when unknown namespace",
			scopeKeys: readNamespace(firstClusterID, "unknown ID"),
			results:   []bool{false, false, false, false},
		},
		{
			name:      "deny when empty namespace",
			scopeKeys: readNamespace(firstClusterID, ""),
			results:   []bool{false, false, false, false},
		},
		{
			name: "allow when global scope resource",
			scopeKeys: []sac.ScopeKey{
				sac.AccessModeScopeKey(storage.Access_READ_ACCESS),
				sac.ResourceScopeKey(resources.Integration.Resource),
			},
			results: []bool{false, true},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scc := subScopeChecker
			for i, scopeKey := range tc.scopeKeys {
				if i >= len(tc.results) {
					assert.Panics(t, func() { scc.SubScopeChecker(scopeKey) })
				} else {
					scc = scc.SubScopeChecker(scopeKey)
					expected := tc.results[i]
					assert.Equalf(t, expected, scc.Allowed(), "scope %s, level [%d]", scopeKey, i)
				}
			}
		})
	}
}

func TestEffectiveAccessScope(t *testing.T) {
	t.Parallel()

	clusterEdit := map[string]storage.Access{string(resources.Cluster.Resource): storage.Access_READ_WRITE_ACCESS}

	complianceEdit := map[string]storage.Access{string(resources.Compliance.Resource): storage.Access_READ_WRITE_ACCESS}

	// Note: The scope tree Compactify function relies on cluster names rather than cluster IDs to identify
	// the cluster part. In order to have the scope validation (which relies on Compactify) working,
	// the clusters in the expected trees are identified with their names rather than ID.

	oneClusterEffectiveScope := effectiveaccessscope.FromClustersAndNamespacesMap([]string{firstClusterName}, nil)

	oneNamespaceScopeMap := map[string][]string{
		firstClusterName: {firstNamespaceName},
	}
	oneNamespaceEffectiveScope := effectiveaccessscope.FromClustersAndNamespacesMap(nil, oneNamespaceScopeMap)

	mixedEffectiveScope := effectiveaccessscope.FromClustersAndNamespacesMap([]string{secondClusterName}, oneNamespaceScopeMap)

	tests := []struct {
		name      string
		roles     []permissions.ResolvedRole
		resource  permissions.ResourceWithAccess
		resultEAS *effectiveaccessscope.ScopeTree
	}{
		{
			name:      "Access to all resources (read-write) for unrestricted scope gives unrestricted scope tree for any resource and access (case read cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-write) for unrestricted scope gives unrestricted scope tree for any resource and access (case write cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-write) for unrestricted scope gives unrestricted scope tree for any resource and access (case read namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-write) for unrestricted scope gives unrestricted scope tree for any resource and access (case write namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-only) for unrestricted scope gives unrestricted scope tree for any resource read (case read cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-write) for unrestricted scope gives deny-all scope tree for any resource write (case write cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-only) for unrestricted scope gives unrestricted scope tree for any resource read (case read namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-write) for unrestricted scope gives deny-all scope tree for any resource write (case write namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-only) for deny-all scope gives deny-all scope tree for any resource and access (case read cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, rolePkg.AccessScopeExcludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-write) for deny-all scope gives deny-all scope tree for any resource and access (case write cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, rolePkg.AccessScopeExcludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-only) for deny-all scope gives deny-all scope tree for any resource and access (case read namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, rolePkg.AccessScopeExcludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-write) for deny-all scope gives deny-all scope tree for any resource and access (case write namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, rolePkg.AccessScopeExcludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-only) for nil scope gives deny-all scope tree for any resource and access (case read cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, nil)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-write) for nil scope gives deny-all scope tree for any resource and access (case write cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, nil)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-only) for nil scope gives deny-all scope tree for any resource and access (case read namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, nil)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-write) for nil scope gives deny-all scope tree for any resource and access (case write namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, nil)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-only) for empty scope gives deny-all scope tree for any resource and access (case read cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, &storage.SimpleAccessScope{Id: "empty"})},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-write) for empty scope gives deny-all scope tree for any resource and access (case write cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, &storage.SimpleAccessScope{Id: "empty"})},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-only) for empty scope gives deny-all scope tree for any resource and access (case read namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, &storage.SimpleAccessScope{Id: "empty"})},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to all resources (read-write) for empty scope gives deny-all scope tree for any resource and access (case write namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, &storage.SimpleAccessScope{Id: "empty"})},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to one resource (read-only) for unrestricted scope gives unrestricted scope tree for the resource and any access (case read cluster)",
			roles:     []permissions.ResolvedRole{role(clusterEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to one resource (read-write) for unrestricted scope gives unrestricted scope tree for the resource and any access (case write cluster)",
			roles:     []permissions.ResolvedRole{role(clusterEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to one resource (read-only) for unrestricted scope gives deny-all scope tree for any other resource and any access (case read namespace)",
			roles:     []permissions.ResolvedRole{role(clusterEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to one resource (read-write) for unrestricted scope gives deny-all scope tree for any other resource and any access (case write namespace)",
			roles:     []permissions.ResolvedRole{role(clusterEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to resource (read-only) for unrestricted scope gives unrestricted scope tree for the resource and any access (case read cluster)",
			roles:     []permissions.ResolvedRole{role(complianceEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Compliance),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to resource (read-write) for unrestricted scope gives unrestricted scope tree for the resource and any access (case write cluster)",
			roles:     []permissions.ResolvedRole{role(complianceEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Compliance),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to replaced resource (read-only) for unrestricted scope gives deny-all scope tree for any other resource and any access (case read namespace)",
			roles:     []permissions.ResolvedRole{role(complianceEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to replaced resource (read-write) for unrestricted scope gives deny-all scope tree for any other resource and any access (case write namespace)",
			roles:     []permissions.ResolvedRole{role(complianceEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to any resource (read-only) for cluster scope gives cluster scope tree for any resource and access (case read cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: oneClusterEffectiveScope,
		},
		{
			name:      "Access to any resource (read-write) for cluster scope gives cluster scope tree for any resource and access (case write cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: oneClusterEffectiveScope,
		},
		{
			name:      "Access to any resource (read-only) for cluster scope gives cluster scope tree for any resource and access (case read namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Namespace),
			resultEAS: oneClusterEffectiveScope,
		},
		{
			name:      "Access to any resource (read-write) for cluster scope gives cluster scope tree for any resource and access (case write namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Namespace),
			resultEAS: oneClusterEffectiveScope,
		},
		{
			name:      "Access to any resource (read-only) for namespace scope gives namespace scope tree for any resource and access (case read cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: oneNamespaceEffectiveScope,
		},
		{
			name:      "Access to any resource (read-write) for namespace scope gives namespace scope tree for any resource and access (case write cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: oneNamespaceEffectiveScope,
		},
		{
			name:      "Access to any resource (read-only) for namespace scope gives namespace scope tree for any resource and access (case read namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Namespace),
			resultEAS: oneNamespaceEffectiveScope,
		},
		{
			name:      "Access to any resource (read-write) for namespace scope gives namespace scope tree for any resource and access (case write namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Namespace),
			resultEAS: oneNamespaceEffectiveScope,
		},
		{
			name:      "Access to any resource (read-only) for namespace scope gives namespace scope tree for any resource and read access (case read cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: oneNamespaceEffectiveScope,
		},
		{
			name:      "Access to any resource (read-write) for namespace scope gives deny-all scope tree for any resource and write access (case write cluster)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to any resource (read-only) for namespace scope gives namespace scope tree for any resource and read access (case read namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Namespace),
			resultEAS: oneNamespaceEffectiveScope,
		},
		{
			name:      "Access to any resource (read-write) for namespace scope gives deny-all scope tree for any resource and write access (case write namespace)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to specific resource (read-only) for namespace scope gives namespace scope tree for the resource and any access (case read cluster)",
			roles:     []permissions.ResolvedRole{role(clusterEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: oneNamespaceEffectiveScope,
		},
		{
			name:      "Access to specific resource (read-write) for namespace scope gives namespace scope tree for the resource and any access (case write cluster)",
			roles:     []permissions.ResolvedRole{role(clusterEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: oneNamespaceEffectiveScope,
		},
		{
			name:      "Access to specific resource (read-only) for namespace scope gives deny-all scope tree for any other resource and access (case read namespace)",
			roles:     []permissions.ResolvedRole{role(clusterEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to specific resource (read-write) for namespace scope gives deny-all scope tree for any other resource and access (case write namespace)",
			roles:     []permissions.ResolvedRole{role(clusterEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name: "Access to specific resource (read-only) for mixed scope gives union scope tree for the resource and any access (case read cluster)",
			roles: []permissions.ResolvedRole{
				role(clusterEdit, withAccessTo1Namespace()), role(clusterEdit, withAccessTo2Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: mixedEffectiveScope,
		},
		{
			name: "Access to specific resource (read-write) for mixed scope gives union scope tree for the resource and any access (case write cluster)",
			roles: []permissions.ResolvedRole{
				role(clusterEdit, withAccessTo1Namespace()), role(clusterEdit, withAccessTo2Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: mixedEffectiveScope,
		},
		{
			name: "Access to specific resource (read-only) for mixed scope gives deny-all scope tree for any other resource and access (case read namespace)",
			roles: []permissions.ResolvedRole{
				role(clusterEdit, withAccessTo1Namespace()), role(clusterEdit, withAccessTo2Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Namespace),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to specific resource (read-only) for namespace scope gives namespace scope tree for the resource and any access (case read cluster)",
			roles:     []permissions.ResolvedRole{role(complianceEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Compliance),
			resultEAS: oneNamespaceEffectiveScope,
		},
		{
			name:      "Access to specific resource (read-write) for namespace scope gives namespace scope tree for the resource and any access (case write cluster)",
			roles:     []permissions.ResolvedRole{role(complianceEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Compliance),
			resultEAS: oneNamespaceEffectiveScope,
		},
		{
			name:      "Access to specific resource (read-only) for namespace scope gives deny-all scope tree for any other resource and access (case read namespace)",
			roles:     []permissions.ResolvedRole{role(complianceEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to specific resource (read-write) for namespace scope gives deny-all scope tree for any other resource and access (case write namespace)",
			roles:     []permissions.ResolvedRole{role(complianceEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name: "Access to specific resource (read-only) for mixed scope gives union scope tree for the resource and any access (case read cluster)",
			roles: []permissions.ResolvedRole{
				role(complianceEdit, withAccessTo1Namespace()), role(complianceEdit, withAccessTo2Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Compliance),
			resultEAS: mixedEffectiveScope,
		},
		{
			name: "Access to specific resource (read-write) for mixed scope gives union scope tree for the resource and any access (case write cluster)",
			roles: []permissions.ResolvedRole{
				role(complianceEdit, withAccessTo1Namespace()), role(complianceEdit, withAccessTo2Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Compliance),
			resultEAS: mixedEffectiveScope,
		},
		{
			name: "Access to specific replaced resource (read-only) for mixed scope gives deny-all scope tree for any other resource and access (case read namespace)",
			roles: []permissions.ResolvedRole{
				role(complianceEdit, withAccessTo1Namespace()), role(complianceEdit, withAccessTo2Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Cluster),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to global resource (read-write) for unrestricted scope gives unrestricted scope for the resource and read access (case read admin)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Administration),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to global resource (read-write) for unrestricted scope gives unrestricted scope for the resource and write access (case write admin)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Administration),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to global resource (read-write) for cluster scope gives unrestricted scope for the resource and read access (case read admin)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Administration),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to global resource (read-write) for cluster scope gives unrestricted scope for the resource and write access (case write admin)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Administration),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to global resource (read-write) for namespace scope gives unrestricted scope for the resource and read access (case read admin)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Administration),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to global resource (read-write) for namespace scope gives unrestricted scope for the resource and write access (case write admin)",
			roles:     []permissions.ResolvedRole{role(allResourcesEdit, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Administration),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to global resource (read-only) for unrestricted scope gives unrestricted scope for the resource and read access (case read admin)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Administration),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to global resource (read-only) for unrestricted scope gives unrestricted scope for the resource and write access (case write admin)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, rolePkg.AccessScopeIncludeAll)},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Administration),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to global resource (read-only) for cluster scope gives unrestricted scope for the resource and read access (case read admin)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Administration),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to global resource (read-only) for cluster scope gives deny-all scope for the resource and write access (case write admin)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Cluster())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Administration),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
		{
			name:      "Access to global resource (read-only) for namespace scope gives unrestricted scope for the resource and read access (case read admin)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_ACCESS, resources.Administration),
			resultEAS: effectiveaccessscope.UnrestrictedEffectiveAccessScope(),
		},
		{
			name:      "Access to global resource (read-only) for namespace scope gives deny-all scope for the resource and write access (case write admin)",
			roles:     []permissions.ResolvedRole{role(allResourcesView, withAccessTo1Namespace())},
			resource:  resourceWithAccess(storage.Access_READ_WRITE_ACCESS, resources.Administration),
			resultEAS: effectiveaccessscope.DenyAllEffectiveAccessScope(),
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scc := newGlobalScopeCheckerCore(clusters, namespaces, tc.roles, nil)
			// Checks on the global level SCC scope extraction
			checkEffectiveAccessScope(t, scc, tc.resource, tc.resultEAS)
			// Checks on the access mode level SCC scope extraction
			scc = scc.SubScopeChecker(sac.AccessModeScopeKey(tc.resource.Access))
			checkEffectiveAccessScope(t, scc, tc.resource, tc.resultEAS)
			// Checks on the (access, resource) level SCC scope extraction
			scc = scc.SubScopeChecker(sac.ResourceScopeKey(tc.resource.Resource.GetResource()))
			checkEffectiveAccessScope(t, scc, tc.resource, tc.resultEAS)
		})
	}
}

func checkEffectiveAccessScope(t *testing.T, scc sac.ScopeCheckerCore, resource permissions.ResourceWithAccess, expectedEas *effectiveaccessscope.ScopeTree) {
	actualEas, err := scc.EffectiveAccessScope(resource)
	assert.Nil(t, err)
	compactExpectedEAS := expectedEas.Compactify()
	compactActualEAS := actualEas.Compactify()
	assert.Equal(t, len(compactExpectedEAS), len(compactActualEAS))
	for clusterID := range compactExpectedEAS {
		assert.ElementsMatch(t, compactExpectedEAS[clusterID], compactActualEAS[clusterID])
	}
}

func TestGlobalScopeCheckerCore(t *testing.T) {
	t.Parallel()
	scc := newGlobalScopeCheckerCore(nil, nil, nil, nil)
	assert.Equal(t, false, scc.Allowed())
}

func TestBuiltInScopeAuthorizerPanicsWhenErrorOnComputeAccessScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		roles     []permissions.ResolvedRole
		scopeKeys []sac.ScopeKey
		results   []bool
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
			scopeKeys: readCluster(firstClusterID, resources.Cluster.Resource),
			results:   []bool{false},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scc := newGlobalScopeCheckerCore(clusters, namespaces, tc.roles, nil)
			for i, scopeKey := range tc.scopeKeys {
				scc = scc.SubScopeChecker(scopeKey)
				if i >= len(tc.results) {
					if !buildinfo.ReleaseBuild {
						assert.Panics(t, func() { scc.Allowed() })
					} else {
						assert.Equal(t, false, scc.Allowed())
					}
				} else {
					expected := tc.results[i]
					assert.Equal(t, expected, scc.Allowed())
				}
			}
		})
	}
}

func readCluster(clusterID string, resource permissions.Resource) []sac.ScopeKey {
	return scopeKeys(storage.Access_READ_ACCESS, resource, clusterID, "")[:3]
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
			IncludedClusters: []string{firstClusterName},
		},
	}
}

func withAccessTo2Cluster() *storage.SimpleAccessScope {
	return &storage.SimpleAccessScope{
		Id: "withAccessTo2Cluster",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{secondClusterName},
		},
	}
}

func withAccessTo1Namespace() *storage.SimpleAccessScope {
	return &storage.SimpleAccessScope{
		Id: "withAccessTo1Namespace",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{
				ClusterName:   firstClusterName,
				NamespaceName: firstNamespaceName,
			}},
		},
	}
}

func resourceWithAccess(access storage.Access, resource permissions.ResourceMetadata) permissions.ResourceWithAccess {
	return permissions.ResourceWithAccess{
		Access:   access,
		Resource: resource,
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

func countAllowedResults(xs []bool) int {
	result := 0
	for _, x := range xs {
		if x == true {
			result++
		}
	}
	return result
}

// region SAC helpers

type clusterForSAC struct {
	ID     string
	Name   string
	Labels map[string]string
}

func (c *clusterForSAC) GetID() string {
	if c == nil {
		return ""
	}
	return c.ID
}

func (c *clusterForSAC) GetName() string {
	if c == nil {
		return ""
	}
	return c.Name
}

func (c *clusterForSAC) GetLabels() map[string]string {
	if c == nil {
		return nil
	}
	return c.Labels
}

type namespaceForSAC struct {
	ID          string
	Name        string
	ClusterID   string
	ClusterName string
	Labels      map[string]string
}

func (n *namespaceForSAC) GetID() string {
	if n == nil {
		return ""
	}
	return n.ID
}

func (n *namespaceForSAC) GetName() string {
	if n == nil {
		return ""
	}
	return n.Name
}

func (n *namespaceForSAC) GetClusterName() string {
	if n == nil {
		return ""
	}
	return n.ClusterName
}

func (n *namespaceForSAC) GetLabels() map[string]string {
	if n == nil {
		return nil
	}
	return n.Labels
}

// endregion SAC helpers
