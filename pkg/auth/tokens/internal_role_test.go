package tokens

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
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
				Permissions: map[AccessWrapper][]string{
					AccessWrapper(storage.Access_READ_ACCESS): make([]string, 0),
				},
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Input with empty write permissions": {
			role: &InternalRole{
				Permissions: map[AccessWrapper][]string{
					AccessWrapper(storage.Access_READ_WRITE_ACCESS): make([]string, 0),
				},
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Input with empty read and write permissions": {
			role: &InternalRole{
				Permissions: map[AccessWrapper][]string{
					AccessWrapper(storage.Access_READ_ACCESS):       make([]string, 0),
					AccessWrapper(storage.Access_READ_WRITE_ACCESS): make([]string, 0),
				},
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Input with nil read permissions": {
			role: &InternalRole{
				Permissions: map[AccessWrapper][]string{
					AccessWrapper(storage.Access_READ_ACCESS): nil,
				},
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Input with nil write permissions": {
			role: &InternalRole{
				Permissions: map[AccessWrapper][]string{
					AccessWrapper(storage.Access_READ_WRITE_ACCESS): nil,
				},
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Input with nil read and write permissions": {
			role: &InternalRole{
				Permissions: map[AccessWrapper][]string{
					AccessWrapper(storage.Access_READ_ACCESS):       nil,
					AccessWrapper(storage.Access_READ_WRITE_ACCESS): nil,
				},
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Single permission - unknown -> no access": {
			role: &InternalRole{
				Permissions: map[AccessWrapper][]string{
					AccessWrapper(storage.Access(-1)): {deploymentResource},
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_NO_ACCESS,
			},
		},
		"Single permission - no access": {
			role: &InternalRole{
				Permissions: map[AccessWrapper][]string{
					AccessWrapper(storage.Access_NO_ACCESS): {deploymentResource},
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_NO_ACCESS,
			},
		},
		"Single permission - read only": {
			role: &InternalRole{
				Permissions: map[AccessWrapper][]string{
					AccessWrapper(storage.Access_READ_ACCESS): {deploymentResource},
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_READ_ACCESS,
			},
		},
		"Single permission - read-write": {
			role: &InternalRole{
				Permissions: map[AccessWrapper][]string{
					AccessWrapper(storage.Access_READ_WRITE_ACCESS): {deploymentResource},
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_READ_WRITE_ACCESS,
			},
		},
		"Multiple permissions - read only": {
			role: &InternalRole{
				Permissions: map[AccessWrapper][]string{
					AccessWrapper(storage.Access_READ_ACCESS): {deploymentResource, imageResource},
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_READ_ACCESS,
				imageResource:      storage.Access_READ_ACCESS,
			},
		},
		"Multiple permissions - read and write": {
			role: &InternalRole{
				Permissions: map[AccessWrapper][]string{
					AccessWrapper(storage.Access_READ_ACCESS):       {deploymentResource},
					AccessWrapper(storage.Access_READ_WRITE_ACCESS): {imageResource},
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
	const clusterName1 = "Cluster1"
	const clusterName2 = "Cluster2"
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
				ClustersByName: make(ClusterScopes),
			},
			expectedScope: emptyScope,
		},
		"Input with one cluster but no access": {
			role: &InternalRole{
				ClustersByName: ClusterScopes{
					clusterName1: []string{},
				},
			},
			expectedScope: emptyScope,
		},
		"Input with multiple clusters but no access": {
			role: &InternalRole{
				ClustersByName: ClusterScopes{
					clusterName1: []string{},
					clusterName2: nil,
				},
			},
			expectedScope: emptyScope,
		},
		"Input with one cluster and full access": {
			role: &InternalRole{
				ClustersByName: ClusterScopes{
					clusterName1: []string{"*"},
				},
			},
			expectedScope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters:   []string{clusterName1},
					IncludedNamespaces: make([]*storage.SimpleAccessScope_Rules_Namespace, 0),
				},
			},
		},
		"Input with single namespace access": {
			role: &InternalRole{
				ClustersByName: ClusterScopes{
					clusterName1: []string{namespaceB},
				},
			},
			expectedScope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters: make([]string, 0),
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterName:   clusterName1,
							NamespaceName: namespaceB,
						},
					},
				},
			},
		},
		"Multiple clusters with access level mix": {
			role: &InternalRole{
				ClustersByName: ClusterScopes{
					clusterName1: []string{namespaceB, namespaceA},
					clusterName2: []string{"*"},
				},
			},
			expectedScope: &storage.SimpleAccessScope{
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters: []string{clusterName2},
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterName:   clusterName1,
							NamespaceName: namespaceA,
						},
						{
							ClusterName:   clusterName1,
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
