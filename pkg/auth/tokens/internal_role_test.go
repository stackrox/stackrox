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
	const readAccess = "READ_ACCESS"
	const invalidAccess = "RANDOM_VALUE_ACCESS"
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
		"Input with empty permissions": {
			role: &InternalRole{
				Permissions: make(map[string]string),
			},
			expectedPermissions: make(map[string]storage.Access),
		},
		"Single permission": {
			role: &InternalRole{
				Permissions: map[string]string{
					deploymentResource: readAccess,
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_READ_ACCESS,
			},
		},
		"Multiple permissions": {
			role: &InternalRole{
				Permissions: map[string]string{
					deploymentResource: readAccess,
					imageResource:      readAccess,
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_READ_ACCESS,
				imageResource:      storage.Access_READ_ACCESS,
			},
		},
		"Unknown access value defaults to no access": {
			role: &InternalRole{
				Permissions: map[string]string{
					deploymentResource: invalidAccess,
				},
			},
			expectedPermissions: map[string]storage.Access{
				deploymentResource: storage.Access_NO_ACCESS,
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
				ClusterScopes: make([]*ClusterScope, 0),
			},
			expectedScope: emptyScope,
		},
		"Input with one cluster but no access": {
			role: &InternalRole{
				ClusterScopes: []*ClusterScope{
					{ClusterName: clusterName1},
				},
			},
			expectedScope: emptyScope,
		},
		"Input with multiple clusters but no access": {
			role: &InternalRole{
				ClusterScopes: []*ClusterScope{
					{ClusterName: clusterName1},
					{ClusterName: clusterName2},
				},
			},
			expectedScope: emptyScope,
		},
		"Input with one cluster and full access": {
			role: &InternalRole{
				ClusterScopes: []*ClusterScope{
					{
						ClusterName:       clusterName1,
						ClusterFullAccess: true,
					},
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
				ClusterScopes: []*ClusterScope{
					{
						ClusterName: clusterName1,
						Namespaces:  []string{namespaceB},
					},
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
		"Nil cluster scopes are ignored": {
			role: &InternalRole{
				ClusterScopes: []*ClusterScope{
					nil,
					{
						ClusterName: clusterName1,
						Namespaces:  []string{namespaceB},
					},
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
		"Cluster scopes without cluster name are ignored": {
			role: &InternalRole{
				ClusterScopes: []*ClusterScope{
					{
						ClusterFullAccess: true,
					},
					{
						ClusterName: clusterName1,
						Namespaces:  []string{namespaceB},
					},
					{
						Namespaces: []string{namespaceA},
					},
					nil,
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
				ClusterScopes: []*ClusterScope{
					{
						ClusterName: clusterName1,
						Namespaces:  []string{namespaceA, namespaceB},
					},
					{
						ClusterName:       clusterName2,
						ClusterFullAccess: true,
					},
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
