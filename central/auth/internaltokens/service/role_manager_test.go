package service

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	clusterDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	roleDataStoreMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/protomock"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCreatePermissionSet(t *testing.T) {
	nilPermissions := (map[string]v1.Access)(nil)
	requestWithoutPermissions := &v1.GenerateTokenForPermissionsAndScopeRequest{
		Permissions: nilPermissions,
	}
	noPermissions := make(map[string]v1.Access)
	requestForNoPermissions := &v1.GenerateTokenForPermissionsAndScopeRequest{
		Permissions: noPermissions,
	}
	onePermission := map[string]v1.Access{
		"Deployment": v1.Access_READ_ACCESS,
	}
	requestForOnePermission := &v1.GenerateTokenForPermissionsAndScopeRequest{
		Permissions: onePermission,
	}
	manyPermissions := map[string]v1.Access{
		"Deployment":   v1.Access_READ_ACCESS,
		"Namespace":    v1.Access_READ_ACCESS,
		"NetworkGraph": v1.Access_READ_ACCESS,
	}
	requestForManyPermissions := &v1.GenerateTokenForPermissionsAndScopeRequest{
		Permissions: manyPermissions,
	}
	for name, tc := range map[string]struct {
		input                 *v1.GenerateTokenForPermissionsAndScopeRequest
		expectedPermissionSet *storage.PermissionSet
		expectedStoreError    error
	}{
		"nil request, successful storage (no access permissions)": {
			input:                 nil,
			expectedPermissionSet: testPermissionSet(nilPermissions),
			expectedStoreError:    nil,
		},
		"nil request, failed storage (no access permissions)": {
			input:                 nil,
			expectedPermissionSet: testPermissionSet(nilPermissions),
			expectedStoreError:    errDummy,
		},
		"request with nil permissions, successful storage (no access permissions)": {
			input:                 requestWithoutPermissions,
			expectedPermissionSet: testPermissionSet(nilPermissions),
			expectedStoreError:    nil,
		},
		"request with nil permissions, failed storage (no access permissions)": {
			input:                 requestWithoutPermissions,
			expectedPermissionSet: testPermissionSet(nilPermissions),
			expectedStoreError:    errDummy,
		},
		"request for no permissions, successful storage (no access permissions)": {
			input:                 requestForNoPermissions,
			expectedPermissionSet: testPermissionSet(noPermissions),
			expectedStoreError:    nil,
		},
		"request for no permissions, failed storage (no access permissions)": {
			input:                 requestForNoPermissions,
			expectedPermissionSet: testPermissionSet(noPermissions),
			expectedStoreError:    errDummy,
		},
		"request for one permissions, successful storage": {
			input:                 requestForOnePermission,
			expectedPermissionSet: testPermissionSet(onePermission),
			expectedStoreError:    nil,
		},
		"request for one permissions, failed storage": {
			input:                 requestForOnePermission,
			expectedPermissionSet: testPermissionSet(onePermission),
			expectedStoreError:    errDummy,
		},
		"request for many permissions, successful storage": {
			input:                 requestForManyPermissions,
			expectedPermissionSet: testPermissionSet(manyPermissions),
			expectedStoreError:    nil,
		},
		"request for many permissions, failed storage": {
			input:                 requestForManyPermissions,
			expectedPermissionSet: testPermissionSet(manyPermissions),
			expectedStoreError:    errDummy,
		},
	} {
		t.Run(name, func(it *testing.T) {
			ctx := sac.WithAllAccess(it.Context())
			mockCtrl := gomock.NewController(it)
			mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
			roleMgr := &roleManager{
				roleStore: mockRoleStore,
			}
			mockRoleStore.EXPECT().
				UpsertPermissionSet(gomock.Any(), protomock.GoMockMatcherEqualMessage(tc.expectedPermissionSet)).
				Times(1).
				Return(tc.expectedStoreError)

			psID, err := roleMgr.createPermissionSet(ctx, tc.input)

			if tc.expectedStoreError != nil {
				assert.Empty(it, psID)
				assert.ErrorIs(it, err, tc.expectedStoreError)
			} else {
				assert.Equal(it, tc.expectedPermissionSet.GetId(), psID)
				assert.NoError(it, err)
			}
		})
	}
}

func TestCreateAccessScope(t *testing.T) {
	targetCluster1 := "cluster 1"
	targetCluster2 := "cluster 2"
	targetCluster3 := "cluster 3"
	targetNamespaceA := "namespace A"
	targetNamespaceB := "namespace B"
	targetNamespaceC := "namespace C"
	requestFullCluster := &v1.ClusterScope{
		ClusterId:         targetCluster1,
		FullClusterAccess: true,
	}
	requestSingleNamespace := &v1.ClusterScope{
		ClusterId:         targetCluster2,
		FullClusterAccess: false,
		Namespaces:        []string{targetNamespaceA},
	}
	requestMultipleNamespaces := &v1.ClusterScope{
		ClusterId:         targetCluster3,
		FullClusterAccess: false,
		Namespaces:        []string{targetNamespaceB, targetNamespaceC},
	}
	for name, tc := range map[string]struct {
		input               *v1.GenerateTokenForPermissionsAndScopeRequest
		expectedAccessScope *storage.SimpleAccessScope
		expectedStoreError  error
	}{
		"nil input, successful storage (empty scope)": {
			input:               nil,
			expectedAccessScope: testAccessScope(nil),
			expectedStoreError:  nil,
		},
		"nil input, failed storage (empty scope)": {
			input:               nil,
			expectedAccessScope: testAccessScope(nil),
			expectedStoreError:  errDummy,
		},
		"input with nil scope, successful storage (empty scope)": {
			input:               &v1.GenerateTokenForPermissionsAndScopeRequest{},
			expectedAccessScope: testAccessScope(nil),
			expectedStoreError:  nil,
		},
		"input with nil scope, failed storage (empty scope)": {
			input:               &v1.GenerateTokenForPermissionsAndScopeRequest{},
			expectedAccessScope: testAccessScope(nil),
			expectedStoreError:  errDummy,
		},
		"input with empty scope, successful storage (empty scope)": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: make([]*v1.ClusterScope, 0),
			},
			expectedAccessScope: testAccessScope(make([]*v1.ClusterScope, 0)),
			expectedStoreError:  nil,
		},
		"input with empty scope, failed storage (empty scope)": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: make([]*v1.ClusterScope, 0),
			},
			expectedAccessScope: testAccessScope(make([]*v1.ClusterScope, 0)),
			expectedStoreError:  errDummy,
		},
		"input with single full cluster scope, successful storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*v1.ClusterScope{requestFullCluster},
			},
			expectedAccessScope: testAccessScope([]*v1.ClusterScope{requestFullCluster}),
			expectedStoreError:  nil,
		},
		"input with single full cluster scope, failed storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*v1.ClusterScope{requestFullCluster},
			},
			expectedAccessScope: testAccessScope([]*v1.ClusterScope{requestFullCluster}),
			expectedStoreError:  errDummy,
		},
		"input with single namespace scope, successful storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			},
			expectedAccessScope: testAccessScope([]*v1.ClusterScope{requestSingleNamespace}),
			expectedStoreError:  nil,
		},
		"input with single namespace scope, failed storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			},
			expectedAccessScope: testAccessScope([]*v1.ClusterScope{requestSingleNamespace}),
			expectedStoreError:  errDummy,
		},
		"input with multi namespace scope, successful storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*v1.ClusterScope{requestMultipleNamespaces},
			},
			expectedAccessScope: testAccessScope([]*v1.ClusterScope{requestMultipleNamespaces}),
			expectedStoreError:  nil,
		},
		"input with multi namespace scope, failed storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*v1.ClusterScope{requestMultipleNamespaces},
			},
			expectedAccessScope: testAccessScope([]*v1.ClusterScope{requestMultipleNamespaces}),
			expectedStoreError:  errDummy,
		},
		"input with multi cluster-namespace scope, successful storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*v1.ClusterScope{
					requestSingleNamespace,
					requestMultipleNamespaces,
				},
			},
			expectedAccessScope: testAccessScope([]*v1.ClusterScope{
				requestSingleNamespace,
				requestMultipleNamespaces,
			}),
			expectedStoreError: nil,
		},
		"input with multi cluster-namespace scope, failed storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*v1.ClusterScope{
					requestSingleNamespace,
					requestMultipleNamespaces,
				},
			},
			expectedAccessScope: testAccessScope([]*v1.ClusterScope{
				requestSingleNamespace,
				requestMultipleNamespaces,
			}),
			expectedStoreError: errDummy,
		},
		"input with complex scope mix, successful storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*v1.ClusterScope{
					requestFullCluster,
					requestSingleNamespace,
					requestMultipleNamespaces,
				},
			},
			expectedAccessScope: testAccessScope([]*v1.ClusterScope{
				requestFullCluster,
				requestSingleNamespace,
				requestMultipleNamespaces,
			}),
			expectedStoreError: nil,
		},
		"input with complex scope mix, failed storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*v1.ClusterScope{
					requestFullCluster,
					requestSingleNamespace,
					requestMultipleNamespaces,
				},
			},
			expectedAccessScope: testAccessScope([]*v1.ClusterScope{
				requestFullCluster,
				requestSingleNamespace,
				requestMultipleNamespaces,
			}),
			expectedStoreError: errDummy,
		},
	} {
		t.Run(name, func(it *testing.T) {
			ctx := sac.WithAllAccess(it.Context())
			mockCtrl := gomock.NewController(it)
			mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
			mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
			roleMgr := &roleManager{
				clusterStore: mockClusterStore,
				roleStore:    mockRoleStore,
			}
			setClusterStoreExpectations(tc.input, mockClusterStore)
			mockRoleStore.EXPECT().
				UpsertAccessScope(gomock.Any(), protomock.GoMockMatcherEqualMessage(tc.expectedAccessScope)).
				Times(1).
				Return(tc.expectedStoreError)

			asID, err := roleMgr.createAccessScope(ctx, tc.input)

			if tc.expectedStoreError != nil {
				assert.Empty(it, asID)
				assert.ErrorIs(it, err, tc.expectedStoreError)
			} else {
				assert.Equal(it, tc.expectedAccessScope.GetId(), asID)
				assert.NoError(it, err)
			}
		})
	}
	t.Run("Cluster name lookup errors are propagated", func(it *testing.T) {
		ctx := sac.WithAllAccess(it.Context())
		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		roleMgr := &roleManager{
			clusterStore: mockClusterStore,
			roleStore:    mockRoleStore,
		}
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			ClusterScopes: []*v1.ClusterScope{requestFullCluster},
		}
		mockClusterStore.EXPECT().
			GetClusterName(gomock.Any(), requestFullCluster.GetClusterId()).
			Times(1).
			Return("", false, errDummy)

		accessScopeId, err := roleMgr.createAccessScope(ctx, input)
		assert.Empty(it, accessScopeId)
		assert.ErrorIs(it, err, errDummy)
	})
	t.Run("Cluster name lookup misses are excluded from resulting scope", func(it *testing.T) {
		ctx := sac.WithAllAccess(it.Context())
		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		roleMgr := &roleManager{
			clusterStore: mockClusterStore,
			roleStore:    mockRoleStore,
		}
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			ClusterScopes: []*v1.ClusterScope{requestFullCluster, requestSingleNamespace},
		}

		expectedAccessScope := testAccessScope([]*v1.ClusterScope{nil, requestSingleNamespace})
		mockClusterStore.EXPECT().
			GetClusterName(gomock.Any(), requestFullCluster.GetClusterId()).
			Times(1).
			Return("", false, nil)
		mockClusterStore.EXPECT().
			GetClusterName(gomock.Any(), requestSingleNamespace.GetClusterId()).
			Times(1).
			Return(requestSingleNamespace.GetClusterId(), true, nil)
		mockRoleStore.EXPECT().
			UpsertAccessScope(gomock.Any(), protomock.GoMockMatcherEqualMessage(expectedAccessScope)).
			Times(1).
			Return(nil)

		accessScopeId, err := roleMgr.createAccessScope(ctx, input)
		assert.Equal(it, expectedAccessScope.GetId(), accessScopeId)
		assert.NoError(it, err)
	})
}

func TestCreateRole(t *testing.T) {
	noPermission := make(map[string]v1.Access)
	deploymentPermission := map[string]v1.Access{
		"Deployment": v1.Access_READ_ACCESS,
	}
	targetCluster1 := "cluster 1"
	targetCluster2 := "cluster 2"
	targetNamespaceA := "namespace A"
	requestFullCluster := &v1.ClusterScope{
		ClusterId:         targetCluster1,
		FullClusterAccess: true,
	}
	requestSingleNamespace := &v1.ClusterScope{
		ClusterId:         targetCluster2,
		FullClusterAccess: false,
		Namespaces:        []string{targetNamespaceA},
	}
	for name, tc := range map[string]struct {
		input                  *v1.GenerateTokenForPermissionsAndScopeRequest
		expectedPermissionSet  *storage.PermissionSet
		expectedAccessScope    *storage.SimpleAccessScope
		expectedRole           *storage.Role
		expectedRoleStoreError error
	}{
		"nil input, successful storage (role with no permission and empty scope)": {
			input:                  nil,
			expectedPermissionSet:  testPermissionSet(noPermission),
			expectedAccessScope:    testAccessScope(make([]*v1.ClusterScope, 0)),
			expectedRole:           testRole(noPermission, make([]*v1.ClusterScope, 0)),
			expectedRoleStoreError: nil,
		},
		"nil input, failed storage (role with no permission and empty scope)": {
			input:                  nil,
			expectedPermissionSet:  testPermissionSet(noPermission),
			expectedAccessScope:    testAccessScope(make([]*v1.ClusterScope, 0)),
			expectedRole:           testRole(noPermission, make([]*v1.ClusterScope, 0)),
			expectedRoleStoreError: errDummy,
		},
		"request for single full cluster access to deployments, successful storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   deploymentPermission,
				ClusterScopes: []*v1.ClusterScope{requestFullCluster},
			},
			expectedPermissionSet:  testPermissionSet(deploymentPermission),
			expectedAccessScope:    testAccessScope([]*v1.ClusterScope{requestFullCluster}),
			expectedRole:           testRole(deploymentPermission, []*v1.ClusterScope{requestFullCluster}),
			expectedRoleStoreError: nil,
		},
		"request for single full cluster access to deployments, failed storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   deploymentPermission,
				ClusterScopes: []*v1.ClusterScope{requestFullCluster},
			},
			expectedPermissionSet:  testPermissionSet(deploymentPermission),
			expectedAccessScope:    testAccessScope([]*v1.ClusterScope{requestFullCluster}),
			expectedRole:           testRole(deploymentPermission, []*v1.ClusterScope{requestFullCluster}),
			expectedRoleStoreError: errDummy,
		},
		"request for single namespace access to deployments, successful storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   deploymentPermission,
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			},
			expectedPermissionSet:  testPermissionSet(deploymentPermission),
			expectedAccessScope:    testAccessScope([]*v1.ClusterScope{requestSingleNamespace}),
			expectedRole:           testRole(deploymentPermission, []*v1.ClusterScope{requestSingleNamespace}),
			expectedRoleStoreError: nil,
		},
		"request for single namespace access to deployments, failed storage": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   deploymentPermission,
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			},
			expectedPermissionSet:  testPermissionSet(deploymentPermission),
			expectedAccessScope:    testAccessScope([]*v1.ClusterScope{requestSingleNamespace}),
			expectedRole:           testRole(deploymentPermission, []*v1.ClusterScope{requestSingleNamespace}),
			expectedRoleStoreError: errDummy,
		},
	} {
		t.Run(name, func(it *testing.T) {
			ctx := sac.WithAllAccess(it.Context())
			mockCtrl := gomock.NewController(it)
			mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
			mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
			roleMgr := &roleManager{
				clusterStore: mockClusterStore,
				roleStore:    mockRoleStore,
			}
			setClusterStoreExpectations(tc.input, mockClusterStore)
			setNormalRoleStoreExpectations(
				tc.expectedPermissionSet,
				tc.expectedAccessScope,
				tc.expectedRole,
				tc.expectedRoleStoreError,
				mockRoleStore,
			)

			roleName, err := roleMgr.createRole(ctx, tc.input)

			if tc.expectedRoleStoreError != nil {
				assert.Empty(it, roleName)
				assert.ErrorIs(it, err, tc.expectedRoleStoreError)
			} else {
				assert.Equal(it, tc.expectedRole.GetName(), roleName)
				assert.NoError(it, err)
			}
		})
	}

	t.Run("access scope creation failure", func(it *testing.T) {
		ctx := sac.WithAllAccess(it.Context())
		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		roleMgr := &roleManager{
			clusterStore: mockClusterStore,
			roleStore:    mockRoleStore,
		}
		expectedPermissionSet := testPermissionSet(deploymentPermission)
		expectedAccessScope := testAccessScope([]*v1.ClusterScope{requestSingleNamespace})
		accessScopeCreationErr := errors.New("access scope creation error")
		mockRoleStore.EXPECT().
			UpsertPermissionSet(gomock.Any(), protomock.GoMockMatcherEqualMessage(expectedPermissionSet)).
			Times(1).
			Return(nil)
		mockRoleStore.EXPECT().
			UpsertAccessScope(gomock.Any(), protomock.GoMockMatcherEqualMessage(expectedAccessScope)).
			Times(1).
			Return(accessScopeCreationErr)

		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   deploymentPermission,
			ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
		}

		setClusterStoreExpectations(input, mockClusterStore)

		roleName, err := roleMgr.createRole(ctx, input)

		assert.Empty(it, roleName)
		assert.ErrorIs(it, err, accessScopeCreationErr)
	})

	t.Run("permission set creation failure", func(it *testing.T) {
		ctx := sac.WithAllAccess(it.Context())
		mockCtrl := gomock.NewController(it)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		roleMgr := &roleManager{
			roleStore: mockRoleStore,
		}
		expectedPermissionSet := testPermissionSet(deploymentPermission)
		permissionSetCreationErr := errors.New("permission set creation error")
		mockRoleStore.EXPECT().
			UpsertPermissionSet(gomock.Any(), protomock.GoMockMatcherEqualMessage(expectedPermissionSet)).
			Times(1).
			Return(permissionSetCreationErr)

		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   deploymentPermission,
			ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
		}

		roleName, err := roleMgr.createRole(ctx, input)

		assert.Empty(it, roleName)
		assert.ErrorIs(it, err, permissionSetCreationErr)
	})
}

func TestConvertAccess(t *testing.T) {
	for name, tc := range map[string]struct {
		in  v1.Access
		out storage.Access
	}{
		v1.Access_NO_ACCESS.String(): {
			in:  v1.Access_NO_ACCESS,
			out: storage.Access_NO_ACCESS,
		},
		v1.Access_READ_ACCESS.String(): {
			in:  v1.Access_READ_ACCESS,
			out: storage.Access_READ_ACCESS,
		},
		v1.Access_READ_WRITE_ACCESS.String(): {
			in:  v1.Access_READ_WRITE_ACCESS,
			out: storage.Access_READ_WRITE_ACCESS,
		},
		"Out of range -> default no access": {
			in:  v1.Access(-1),
			out: storage.Access_NO_ACCESS,
		},
	} {
		t.Run(name, func(it *testing.T) {
			converted := convertAccess(tc.in)
			assert.Equal(it, tc.out, converted)
		})
	}
}

func computePermissionSetID(permissions map[string]v1.Access) string {
	resources := make([]string, 0, len(permissions))
	for res := range permissions {
		resources = append(resources, res)
	}
	slices.Sort(resources)
	individualPermissions := make([]string, 0, len(resources))
	for _, resource := range resources {
		access := permissions[resource]
		individualPermissions = append(
			individualPermissions,
			fmt.Sprintf("%s%s%s", resource, keyValueSeparator, access.String()),
		)
	}
	permissionString := strings.Join(individualPermissions, primaryListSeparator)
	return declarativeconfig.NewDeclarativePermissionSetUUID(permissionString).String()
}

func computeAccessScopeID(targetScopes []*v1.ClusterScope) string {
	clusterScopes := make([]string, 0, len(targetScopes))
	for _, targetScope := range targetScopes {
		if targetScope == nil {
			clusterScopes = append(clusterScopes, "")
			continue
		}
		var namespaceScope string
		if targetScope.GetFullClusterAccess() {
			namespaceScope = clusterWildCard
		} else {
			namespaceScope = strings.Join(targetScope.GetNamespaces(), secondaryListSeparator)
		}
		clusterScopes = append(
			clusterScopes,
			fmt.Sprintf("%s%s%s", targetScope.GetClusterId(), keyValueSeparator, namespaceScope),
		)
	}
	accessScopeString := strings.Join(clusterScopes, primaryListSeparator)
	return declarativeconfig.NewDeclarativeAccessScopeUUID(accessScopeString).String()
}

func computeRoleName(permissions map[string]v1.Access, targetScopes []*v1.ClusterScope) string {
	permissionSetID := computePermissionSetID(permissions)
	accessScopeID := computeAccessScopeID(targetScopes)
	return fmt.Sprintf(roleNameFormat, permissionSetID, accessScopeID)
}

func testPermissionSet(permissions map[string]v1.Access) *storage.PermissionSet {
	resources := make([]string, 0, len(permissions))
	for res := range permissions {
		resources = append(resources, res)
	}
	permissionSetID := computePermissionSetID(permissions)
	permissionSet := &storage.PermissionSet{
		Id:               permissionSetID,
		Name:             fmt.Sprintf(permissionSetNameFormat, permissionSetID),
		ResourceToAccess: make(map[string]storage.Access),
		Traits:           generatedObjectTraits.CloneVT(),
	}
	for _, resource := range resources {
		permissionSet.ResourceToAccess[resource] = convertAccess(permissions[resource])
	}
	return permissionSet
}

func testAccessScope(targetScopes []*v1.ClusterScope) *storage.SimpleAccessScope {
	accessScopeID := computeAccessScopeID(targetScopes)
	accessScope := &storage.SimpleAccessScope{
		Id:   accessScopeID,
		Name: fmt.Sprintf(accessScopeNameFormat, accessScopeID),
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters:   make([]string, 0),
			IncludedNamespaces: make([]*storage.SimpleAccessScope_Rules_Namespace, 0),
		},
		Traits: generatedObjectTraits.CloneVT(),
	}
	for _, targetScope := range targetScopes {
		if targetScope == nil {
			continue
		}
		if targetScope.GetFullClusterAccess() {
			accessScope.Rules.IncludedClusters = append(
				accessScope.Rules.IncludedClusters,
				targetScope.GetClusterId(),
			)
		} else {
			for _, namespace := range targetScope.GetNamespaces() {
				accessScope.Rules.IncludedNamespaces = append(
					accessScope.Rules.IncludedNamespaces,
					&storage.SimpleAccessScope_Rules_Namespace{
						ClusterName:   targetScope.GetClusterId(),
						NamespaceName: namespace,
					},
				)
			}
		}
	}
	return accessScope
}

func testRole(permissions map[string]v1.Access, targetScopes []*v1.ClusterScope) *storage.Role {
	permissionSetID := computePermissionSetID(permissions)
	accessScopeID := computeAccessScopeID(targetScopes)
	role := &storage.Role{
		Name:            computeRoleName(permissions, targetScopes),
		Description:     "Generated role for OCP console plugin",
		PermissionSetId: permissionSetID,
		AccessScopeId:   accessScopeID,
		Traits:          generatedObjectTraits.CloneVT(),
	}
	return role
}

func setClusterStoreExpectations(
	input *v1.GenerateTokenForPermissionsAndScopeRequest,
	mockClusterStore *clusterDataStoreMocks.MockDataStore,
) {
	for _, clusterScope := range input.GetClusterScopes() {
		clusterIdName := clusterScope.GetClusterId()
		mockClusterStore.EXPECT().
			GetClusterName(gomock.Any(), clusterIdName).
			Times(1).
			Return(clusterIdName, true, nil)
	}
}

func setNormalRoleStoreExpectations(
	permissionSet *storage.PermissionSet,
	accessScope *storage.SimpleAccessScope,
	role *storage.Role,
	roleStoreError error,
	mockRoleStore *roleDataStoreMocks.MockDataStore,
) {
	mockRoleStore.EXPECT().
		UpsertPermissionSet(
			gomock.Any(),
			protomock.GoMockMatcherEqualMessage(permissionSet),
		).Times(1).Return(nil)
	mockRoleStore.EXPECT().
		UpsertAccessScope(
			gomock.Any(),
			protomock.GoMockMatcherEqualMessage(accessScope),
		).Times(1).Return(nil)
	mockRoleStore.EXPECT().
		UpsertRole(
			gomock.Any(),
			protomock.GoMockMatcherEqualMessage(role),
		).
		Times(1).Return(roleStoreError)
}
