package tokens

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestInternalRoleGetRoleName(t *testing.T) {
	nilRole := (*InternalRole)(nil)
	assert.Equal(t, "", nilRole.GetRoleName())
	emptyRole := &InternalRole{}
	assert.Equal(t, "", emptyRole.GetRoleName())
	const roleName1 = "role1"
	roleWithName1 := &InternalRole{RoleName: roleName1}
	assert.Equal(t, roleName1, roleWithName1.GetRoleName())
	const roleName2 = "role2"
	roleWithName2 := &InternalRole{RoleName: roleName2}
	assert.Equal(t, roleName2, roleWithName2.GetRoleName())
}

func TestInternalRoleGetPermissions(t *testing.T) {
	const deploymentResource = "Deployment"
	const imageResource = "Image"
	for name, tc := range map[string]struct {
		role                *InternalRole
		expectedPermissions map[string]storage.Access
	}{
		"Nil input": {
			role:                nil,
			expectedPermissions: nil,
		},
		"Empty input": {
			role:                &InternalRole{},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Input with empty read permissions": {
			role: &InternalRole{
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: make([]string, 0),
				},
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Input with empty write permissions": {
			role: &InternalRole{
				Permissions: map[storage.Access][]string{
					storage.Access_READ_WRITE_ACCESS: make([]string, 0),
				},
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Input with empty read and write permissions": {
			role: &InternalRole{
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS:       make([]string, 0),
					storage.Access_READ_WRITE_ACCESS: make([]string, 0),
				},
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Input with nil read permissions": {
			role: &InternalRole{
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: nil,
				},
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Input with nil write permissions": {
			role: &InternalRole{
				Permissions: map[storage.Access][]string{
					storage.Access_READ_WRITE_ACCESS: nil,
				},
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Input with nil read and write permissions": {
			role: &InternalRole{
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS:       nil,
					storage.Access_READ_WRITE_ACCESS: nil,
				},
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Single permission - unknown -> no access": {
			role: &InternalRole{
				Permissions: map[storage.Access][]string{
					storage.Access(-1): {deploymentResource},
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_NO_ACCESS,
			},
		},
		"Single permission - no access": {
			role: &InternalRole{
				Permissions: map[storage.Access][]string{
					storage.Access_NO_ACCESS: {deploymentResource},
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_NO_ACCESS,
			},
		},
		"Single permission - read only": {
			role: &InternalRole{
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {deploymentResource},
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_READ_ACCESS,
			},
		},
		"Single permission - read-write": {
			role: &InternalRole{
				Permissions: map[storage.Access][]string{
					storage.Access_READ_WRITE_ACCESS: {deploymentResource},
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_READ_WRITE_ACCESS,
			},
		},
		"Multiple permissions - read only": {
			role: &InternalRole{
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {deploymentResource, imageResource},
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_READ_ACCESS,
				imageResource:      storage.Access_READ_ACCESS,
			},
		},
		"Multiple permissions - read and write": {
			role: &InternalRole{
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS:       {deploymentResource},
					storage.Access_READ_WRITE_ACCESS: {imageResource},
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_READ_ACCESS,
				imageResource:      storage.Access_READ_WRITE_ACCESS,
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			assert.Equal(it, tc.expectedPermissions, tc.role.GetPermissions())
		})
	}
}

func TestInternalRoleGetAccessScope(t *testing.T) {
	const namespaceA = "namespace-A"
	const namespaceB = "namespace-B"
	emptyScope := &storage.SimpleAccessScope{
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters:   make([]string, 0),
			IncludedNamespaces: make([]*storage.SimpleAccessScope_Rules_Namespace, 0),
		},
	}
	for name, tc := range map[string]struct {
		role          *InternalRole
		expectedScope *storage.SimpleAccessScope
	}{
		"Nil input": {
			role:          nil,
			expectedScope: nil,
		},
		"Empty input": {
			role:          &InternalRole{},
			expectedScope: emptyScope,
		},
		"Input with empty scope": {
			role: &InternalRole{
				Clusters: make(ClusterScopes),
			},
			expectedScope: emptyScope,
		},
		"Input with one cluster but no access": {
			role: &InternalRole{
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {},
				},
			},
			expectedScope: emptyScope,
		},
		"Input with multiple clusters but no access": {
			role: &InternalRole{
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {},
					fixtureconsts.Cluster2: nil,
				},
			},
			expectedScope: emptyScope,
		},
		"Input with one cluster and full access": {
			role: &InternalRole{
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {"*"},
				},
			},
			expectedScope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusterIds: []string{fixtureconsts.Cluster1},
					IncludedNamespaces: make([]*storage.SimpleAccessScope_Rules_Namespace, 0),
				},
			},
		},
		"Input with single namespace access": {
			role: &InternalRole{
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {namespaceB},
				},
			},
			expectedScope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters: make([]string, 0),
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterId:     fixtureconsts.Cluster1,
							NamespaceName: namespaceB,
						},
					},
				},
			},
		},
		"Multiple clusters with access level mix": {
			role: &InternalRole{
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {namespaceB, namespaceA},
					fixtureconsts.Cluster2: {"*"},
				},
			},
			expectedScope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusterIds: []string{fixtureconsts.Cluster2},
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterId:     fixtureconsts.Cluster1,
							NamespaceName: namespaceA,
						},
						{
							ClusterId:     fixtureconsts.Cluster1,
							NamespaceName: namespaceB,
						},
					},
				},
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			protoassert.Equal(it, tc.expectedScope, tc.role.GetAccessScope())
		})
	}
}

func TestInternalRoleJSONEncoding(t *testing.T) {
	const namespaceA = "namespace-a"
	const namespaceB = "namespace-b"
	const deploymentResource = "Deployment"
	const imageResource = "Image"

	for name, tc := range map[string]struct {
		role         *InternalRole
		expectedJSON string
	}{
		"Nil role": {
			role:         nil,
			expectedJSON: "null",
		},
		"Empty role": {
			role:         &InternalRole{},
			expectedJSON: "{}",
		},
		"Role with only name": {
			role: &InternalRole{
				RoleName: "test-role",
			},
			expectedJSON: `{"name":"test-role"}`,
		},
		"Role with read permissions": {
			role: &InternalRole{
				RoleName: "reader-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {deploymentResource, imageResource},
				},
			},
			expectedJSON: `{"name":"reader-role","permissions":{"read":["Deployment","Image"]}}`,
		},
		"Role with read and write permissions": {
			role: &InternalRole{
				RoleName: "admin-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS:       {deploymentResource},
					storage.Access_READ_WRITE_ACCESS: {imageResource},
				},
			},
			expectedJSON: `{"name":"admin-role","permissions":{"read":["Deployment"],"read-write":["Image"]}}`,
		},
		"Role with cluster scopes": {
			role: &InternalRole{
				RoleName: "scoped-role",
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {namespaceA, namespaceB},
					fixtureconsts.Cluster2: {"*"},
				},
			},
			expectedJSON: `{"name":"scoped-role","clusters":{"caaaaaaa-bbbb-4011-0000-111111111111":["namespace-a","namespace-b"],"caaaaaaa-bbbb-4011-0000-222222222222":["*"]}}`,
		},
		"Complete role with all fields": {
			role: &InternalRole{
				RoleName: "complete-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS:       {deploymentResource},
					storage.Access_READ_WRITE_ACCESS: {imageResource},
				},
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {namespaceA},
				},
			},
			expectedJSON: `{"name":"complete-role","permissions":{"read":["Deployment"],"read-write":["Image"]},"clusters":{"caaaaaaa-bbbb-4011-0000-111111111111":["namespace-a"]}}`,
		},
		"Role with NO_ACCESS permission normalizes to NO_ACCESS": {
			role: &InternalRole{
				RoleName: "no-access-role",
				Permissions: map[storage.Access][]string{
					storage.Access_NO_ACCESS: {deploymentResource},
				},
			},
			expectedJSON: `{"name":"no-access-role","permissions":{"none":["Deployment"]}}`,
		},
		"Role with unknown access level normalizes to NO_ACCESS": {
			role: &InternalRole{
				RoleName: "unknown-access-role",
				Permissions: map[storage.Access][]string{
					storage.Access(-1): {deploymentResource},
				},
			},
			expectedJSON: `{"name":"unknown-access-role","permissions":{"none":["Deployment"]}}`,
		},
	} {
		t.Run(name, func(it *testing.T) {
			// Test marshaling
			jsonBytes, err := json.Marshal(tc.role)
			assert.NoError(it, err)

			// Verify JSON structure matches expected (order-independent comparison for maps)
			var actualJSON map[string]interface{}
			var expectedJSON map[string]interface{}

			if tc.role == nil {
				assert.Equal(it, "null", string(jsonBytes))
				return
			}

			err = json.Unmarshal(jsonBytes, &actualJSON)
			assert.NoError(it, err)
			err = json.Unmarshal([]byte(tc.expectedJSON), &expectedJSON)
			assert.NoError(it, err)
			assert.Equal(it, expectedJSON, actualJSON)

			// Test round-trip: unmarshal back and verify it matches original
			var unmarshaledRole InternalRole
			err = json.Unmarshal(jsonBytes, &unmarshaledRole)
			assert.NoError(it, err)
			assert.Equal(it, tc.role.RoleName, unmarshaledRole.RoleName)
			assert.Equal(it, tc.role.GetPermissions(), unmarshaledRole.GetPermissions())
			protoassert.Equal(it, tc.role.GetAccessScope(), unmarshaledRole.GetAccessScope())
		})
	}
}

// mockClusterResolver is a test implementation of ClusterResolver
type mockClusterResolver struct {
	clusters map[string]string // map of cluster name -> cluster ID
	err      error             // error to return from GetClusterID
}

func (m *mockClusterResolver) GetClusterID(ctx context.Context, name string) (string, bool, error) {
	if m.err != nil {
		return "", false, m.err
	}
	id, found := m.clusters[name]
	return id, found, nil
}

func TestNewInternalRoleFromPermissionsAndScope(t *testing.T) {
	const deploymentResource = "Deployment"
	const imageResource = "Image"
	const namespaceA = "namespace-a"
	const namespaceB = "namespace-b"
	const clusterNameDev = "dev-cluster"
	const clusterNameProd = "prod-cluster"

	for name, tc := range map[string]struct {
		roleName      string
		permissions   *storage.PermissionSet
		scope         *storage.SimpleAccessScope
		resolver      ClusterResolver
		expectedRole  *InternalRole
		expectedError string
	}{
		"Nil resolver returns error": {
			roleName: "test-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{},
			},
			resolver:      nil,
			expectedError: "missing cluster ID resolver",
		},
		"Empty permissions and scope": {
			roleName: "empty-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{},
			},
			resolver: &mockClusterResolver{clusters: map[string]string{}},
			expectedRole: &InternalRole{
				RoleName:    "empty-role",
				Permissions: map[storage.Access][]string{},
				Clusters:    ClusterScopes{},
			},
		},
		"Role with read permissions only": {
			roleName: "reader-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
					imageResource:      storage.Access_READ_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{},
			},
			resolver: &mockClusterResolver{clusters: map[string]string{}},
			expectedRole: &InternalRole{
				RoleName: "reader-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {deploymentResource, imageResource},
				},
				Clusters: ClusterScopes{},
			},
		},
		"Role with write permissions only": {
			roleName: "writer-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_WRITE_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{},
			},
			resolver: &mockClusterResolver{clusters: map[string]string{}},
			expectedRole: &InternalRole{
				RoleName: "writer-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_WRITE_ACCESS: {deploymentResource},
				},
				Clusters: ClusterScopes{},
			},
		},
		"Role with mixed read and write permissions": {
			roleName: "mixed-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
					imageResource:      storage.Access_READ_WRITE_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{},
			},
			resolver: &mockClusterResolver{clusters: map[string]string{}},
			expectedRole: &InternalRole{
				RoleName: "mixed-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS:       {deploymentResource},
					storage.Access_READ_WRITE_ACCESS: {imageResource},
				},
				Clusters: ClusterScopes{},
			},
		},
		"Scope with cluster IDs only": {
			roleName: "cluster-id-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusterIds: []string{fixtureconsts.Cluster1, fixtureconsts.Cluster2},
				},
			},
			resolver: &mockClusterResolver{clusters: map[string]string{}},
			expectedRole: &InternalRole{
				RoleName: "cluster-id-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {deploymentResource},
				},
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {"*"},
					fixtureconsts.Cluster2: {"*"},
				},
			},
		},
		"Scope with cluster names that resolve": {
			roleName: "cluster-name-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters: []string{clusterNameDev, clusterNameProd},
				},
			},
			resolver: &mockClusterResolver{
				clusters: map[string]string{
					clusterNameDev:  fixtureconsts.Cluster1,
					clusterNameProd: fixtureconsts.Cluster2,
				},
			},
			expectedRole: &InternalRole{
				RoleName: "cluster-name-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {deploymentResource},
				},
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {"*"},
					fixtureconsts.Cluster2: {"*"},
				},
			},
		},
		"Scope with cluster names that don't resolve are skipped": {
			roleName: "missing-cluster-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters: []string{clusterNameDev, "unknown-cluster"},
				},
			},
			resolver: &mockClusterResolver{
				clusters: map[string]string{
					clusterNameDev: fixtureconsts.Cluster1,
					// "unknown-cluster" not in map
				},
			},
			expectedRole: &InternalRole{
				RoleName: "missing-cluster-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {deploymentResource},
				},
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {"*"},
					// unknown-cluster is skipped
				},
			},
		},
		"Cluster name resolution error propagates": {
			roleName: "error-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters: []string{clusterNameDev},
				},
			},
			resolver: &mockClusterResolver{
				err: errors.New("resolver error"),
			},
			expectedError: "failed to resolve cluster ID",
		},
		"Scope with namespace rules using cluster IDs": {
			roleName: "namespace-id-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterId:     fixtureconsts.Cluster1,
							NamespaceName: namespaceA,
						},
						{
							ClusterId:     fixtureconsts.Cluster1,
							NamespaceName: namespaceB,
						},
					},
				},
			},
			resolver: &mockClusterResolver{clusters: map[string]string{}},
			expectedRole: &InternalRole{
				RoleName: "namespace-id-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {deploymentResource},
				},
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {namespaceA, namespaceB},
				},
			},
		},
		"Scope with namespace rules using cluster names": {
			roleName: "namespace-name-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterName:   clusterNameDev,
							NamespaceName: namespaceA,
						},
						{
							ClusterName:   clusterNameProd,
							NamespaceName: namespaceB,
						},
					},
				},
			},
			resolver: &mockClusterResolver{
				clusters: map[string]string{
					clusterNameDev:  fixtureconsts.Cluster1,
					clusterNameProd: fixtureconsts.Cluster2,
				},
			},
			expectedRole: &InternalRole{
				RoleName: "namespace-name-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {deploymentResource},
				},
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {namespaceA},
					fixtureconsts.Cluster2: {namespaceB},
				},
			},
		},
		"Namespace with unresolved cluster name is skipped": {
			roleName: "namespace-missing-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterName:   clusterNameDev,
							NamespaceName: namespaceA,
						},
						{
							ClusterName:   "unknown-cluster",
							NamespaceName: namespaceB,
						},
					},
				},
			},
			resolver: &mockClusterResolver{
				clusters: map[string]string{
					clusterNameDev: fixtureconsts.Cluster1,
					// "unknown-cluster" not in map
				},
			},
			expectedRole: &InternalRole{
				RoleName: "namespace-missing-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {deploymentResource},
				},
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {namespaceA},
					// namespace with unknown cluster is skipped
				},
			},
		},
		"Namespace cluster name resolution error propagates": {
			roleName: "namespace-error-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterName:   clusterNameDev,
							NamespaceName: namespaceA,
						},
					},
				},
			},
			resolver: &mockClusterResolver{
				err: errors.New("namespace resolver error"),
			},
			expectedError: "failed to resolve cluster ID",
		},
		"Mixed scope with cluster IDs, cluster names, and namespace rules": {
			roleName: "complex-scope-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
					imageResource:      storage.Access_READ_WRITE_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusterIds: []string{fixtureconsts.Cluster1},
					IncludedClusters:   []string{clusterNameProd},
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterId:     fixtureconsts.Cluster3,
							NamespaceName: namespaceA,
						},
					},
				},
			},
			resolver: &mockClusterResolver{
				clusters: map[string]string{
					clusterNameProd: fixtureconsts.Cluster2,
				},
			},
			expectedRole: &InternalRole{
				RoleName: "complex-scope-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS:       {deploymentResource},
					storage.Access_READ_WRITE_ACCESS: {imageResource},
				},
				Clusters: ClusterScopes{
					fixtureconsts.Cluster1: {"*"},
					fixtureconsts.Cluster2: {"*"},
					fixtureconsts.Cluster3: {namespaceA},
				},
			},
		},
		"Namespace appends to existing cluster scope": {
			roleName: "namespace-append-role",
			permissions: &storage.PermissionSet{
				ResourceToAccess: map[string]storage.Access{
					deploymentResource: storage.Access_READ_ACCESS,
				},
			},
			scope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusterIds: []string{fixtureconsts.Cluster1},
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterId:     fixtureconsts.Cluster1,
							NamespaceName: namespaceA,
						},
					},
				},
			},
			resolver: &mockClusterResolver{clusters: map[string]string{}},
			expectedRole: &InternalRole{
				RoleName: "namespace-append-role",
				Permissions: map[storage.Access][]string{
					storage.Access_READ_ACCESS: {deploymentResource},
				},
				Clusters: ClusterScopes{
					// Cluster1 has both wildcard and namespace - appended
					fixtureconsts.Cluster1: {"*", namespaceA},
				},
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			ctx := context.Background()
			role, err := NewInternalRoleFromPermissionsAndScope(
				ctx,
				tc.roleName,
				tc.permissions,
				tc.scope,
				tc.resolver,
			)

			if tc.expectedError != "" {
				assert.Error(it, err)
				assert.Contains(it, err.Error(), tc.expectedError)
				return
			}

			assert.NoError(it, err)
			assert.Equal(it, tc.expectedRole.RoleName, role.RoleName)
			assert.Equal(it, tc.expectedRole.Permissions, role.Permissions)
			assert.Equal(it, tc.expectedRole.Clusters, role.Clusters)
		})
	}
}
