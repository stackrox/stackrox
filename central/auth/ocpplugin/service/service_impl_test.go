package service

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	roleDataStoreMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	tokensMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protomock"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	errDummy = errors.New("test error")

	expectedExpiration = time.Date(1989, time.November, 9, 18, 10, 35, 987654321, time.UTC)
)

func testClock() time.Time {
	return time.Date(1989, time.November, 9, 18, 05, 35, 987654321, time.UTC)
}

func TestCreatePermissionSet(t *testing.T) {
	requestWithoutPermissions := &central.GenerateTokenForPermissionsAndScopeRequest{
		ReadPermissions: nil,
	}
	requestForNoPermissions := &central.GenerateTokenForPermissionsAndScopeRequest{
		ReadPermissions: make([]string, 0),
	}
	onePermission := []string{"Deployment"}
	requestForOnePermission := &central.GenerateTokenForPermissionsAndScopeRequest{
		ReadPermissions: onePermission,
	}
	manyPermissions := []string{"Deployment", "Namespace", "NetworkGraph"}
	requestForManyPermissions := &central.GenerateTokenForPermissionsAndScopeRequest{
		ReadPermissions: manyPermissions,
	}
	for name, tc := range map[string]struct {
		input                 *central.GenerateTokenForPermissionsAndScopeRequest
		expectedPermissionSet *storage.PermissionSet
		expectedStoreError    error
	}{
		"nil request, successful storage (no access permissions)": {
			input:                 nil,
			expectedPermissionSet: testPermissionSet(nil),
			expectedStoreError:    nil,
		},
		"nil request, failed storage (no access permissions)": {
			input:                 nil,
			expectedPermissionSet: testPermissionSet(nil),
			expectedStoreError:    errDummy,
		},
		"request with nil permissions, successful storage (no access permissions)": {
			input:                 requestWithoutPermissions,
			expectedPermissionSet: testPermissionSet(nil),
			expectedStoreError:    nil,
		},
		"request with nil permissions, failed storage (no access permissions)": {
			input:                 requestWithoutPermissions,
			expectedPermissionSet: testPermissionSet(nil),
			expectedStoreError:    errDummy,
		},
		"request for no permissions, successful storage (no access permissions)": {
			input:                 requestForNoPermissions,
			expectedPermissionSet: testPermissionSet(make([]string, 0)),
			expectedStoreError:    nil,
		},
		"request for no permissions, failed storage (no access permissions)": {
			input:                 requestForNoPermissions,
			expectedPermissionSet: testPermissionSet(make([]string, 0)),
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
			svc := &serviceImpl{
				roleStore: mockRoleStore,
			}
			mockRoleStore.EXPECT().
				UpsertPermissionSet(gomock.Any(), protomock.GoMockMatcherEqualMessage(tc.expectedPermissionSet)).
				Times(1).
				Return(tc.expectedStoreError)

			psID, err := svc.createPermissionSet(ctx, tc.input)

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

func computePermissionSetID(readResources []string) string {
	individualPermissions := make([]string, 0, len(readResources))
	for _, resource := range readResources {
		individualPermissions = append(
			individualPermissions,
			fmt.Sprintf("%s%s%s", resource, keyValueSeparator, "READ_ACCESS"),
		)
	}
	permissionString := strings.Join(individualPermissions, primaryListSeparator)
	return declarativeconfig.NewDeclarativePermissionSetUUID(permissionString).String()
}

func testPermissionSet(readResources []string) *storage.PermissionSet {
	permissionSetID := computePermissionSetID(readResources)
	permissionSet := &storage.PermissionSet{
		Id:               permissionSetID,
		Name:             fmt.Sprintf(permissionSetNameFormat, permissionSetID),
		ResourceToAccess: make(map[string]storage.Access),
		Traits:           generatedObjectTraits.CloneVT(),
	}
	for _, resource := range readResources {
		permissionSet.ResourceToAccess[resource] = storage.Access_READ_ACCESS
	}
	return permissionSet
}

func TestCreateAccessScope(t *testing.T) {
	targetCluster1 := "cluster 1"
	targetCluster2 := "cluster 2"
	targetCluster3 := "cluster 3"
	targetNamespaceA := "namespace A"
	targetNamespaceB := "namespace B"
	targetNamespaceC := "namespace C"
	requestFullCluster := &central.RequestedRoleClusterScope{
		ClusterName:       targetCluster1,
		FullClusterAccess: true,
	}
	requestSingleNamespace := &central.RequestedRoleClusterScope{
		ClusterName:       targetCluster2,
		FullClusterAccess: false,
		Namespaces:        []string{targetNamespaceA},
	}
	requestMultipleNamespaces := &central.RequestedRoleClusterScope{
		ClusterName:       targetCluster3,
		FullClusterAccess: false,
		Namespaces:        []string{targetNamespaceB, targetNamespaceC},
	}
	for name, tc := range map[string]struct {
		input               *central.GenerateTokenForPermissionsAndScopeRequest
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
			input:               &central.GenerateTokenForPermissionsAndScopeRequest{},
			expectedAccessScope: testAccessScope(nil),
			expectedStoreError:  nil,
		},
		"input with nil scope, failed storage (empty scope)": {
			input:               &central.GenerateTokenForPermissionsAndScopeRequest{},
			expectedAccessScope: testAccessScope(nil),
			expectedStoreError:  errDummy,
		},
		"input with empty scope, successful storage (empty scope)": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: make([]*central.RequestedRoleClusterScope, 0),
			},
			expectedAccessScope: testAccessScope(make([]*central.RequestedRoleClusterScope, 0)),
			expectedStoreError:  nil,
		},
		"input with empty scope, failed storage (empty scope)": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: make([]*central.RequestedRoleClusterScope, 0),
			},
			expectedAccessScope: testAccessScope(make([]*central.RequestedRoleClusterScope, 0)),
			expectedStoreError:  errDummy,
		},
		"input with single full cluster scope, successful storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*central.RequestedRoleClusterScope{requestFullCluster},
			},
			expectedAccessScope: testAccessScope([]*central.RequestedRoleClusterScope{requestFullCluster}),
			expectedStoreError:  nil,
		},
		"input with single full cluster scope, failed storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*central.RequestedRoleClusterScope{requestFullCluster},
			},
			expectedAccessScope: testAccessScope([]*central.RequestedRoleClusterScope{requestFullCluster}),
			expectedStoreError:  errDummy,
		},
		"input with single namespace scope, successful storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*central.RequestedRoleClusterScope{requestSingleNamespace},
			},
			expectedAccessScope: testAccessScope([]*central.RequestedRoleClusterScope{requestSingleNamespace}),
			expectedStoreError:  nil,
		},
		"input with single namespace scope, failed storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*central.RequestedRoleClusterScope{requestSingleNamespace},
			},
			expectedAccessScope: testAccessScope([]*central.RequestedRoleClusterScope{requestSingleNamespace}),
			expectedStoreError:  errDummy,
		},
		"input with multi namespace scope, successful storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*central.RequestedRoleClusterScope{requestMultipleNamespaces},
			},
			expectedAccessScope: testAccessScope([]*central.RequestedRoleClusterScope{requestMultipleNamespaces}),
			expectedStoreError:  nil,
		},
		"input with multi namespace scope, failed storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*central.RequestedRoleClusterScope{requestMultipleNamespaces},
			},
			expectedAccessScope: testAccessScope([]*central.RequestedRoleClusterScope{requestMultipleNamespaces}),
			expectedStoreError:  errDummy,
		},
		"input with multi cluster-namespace scope, successful storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*central.RequestedRoleClusterScope{
					requestSingleNamespace,
					requestMultipleNamespaces,
				},
			},
			expectedAccessScope: testAccessScope([]*central.RequestedRoleClusterScope{
				requestSingleNamespace,
				requestMultipleNamespaces,
			}),
			expectedStoreError: nil,
		},
		"input with multi cluster-namespace scope, failed storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*central.RequestedRoleClusterScope{
					requestSingleNamespace,
					requestMultipleNamespaces,
				},
			},
			expectedAccessScope: testAccessScope([]*central.RequestedRoleClusterScope{
				requestSingleNamespace,
				requestMultipleNamespaces,
			}),
			expectedStoreError: errDummy,
		},
		"input with complex scope mix, successful storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*central.RequestedRoleClusterScope{
					requestFullCluster,
					requestSingleNamespace,
					requestMultipleNamespaces,
				},
			},
			expectedAccessScope: testAccessScope([]*central.RequestedRoleClusterScope{
				requestFullCluster,
				requestSingleNamespace,
				requestMultipleNamespaces,
			}),
			expectedStoreError: nil,
		},
		"input with complex scope mix, failed storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*central.RequestedRoleClusterScope{
					requestFullCluster,
					requestSingleNamespace,
					requestMultipleNamespaces,
				},
			},
			expectedAccessScope: testAccessScope([]*central.RequestedRoleClusterScope{
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
			mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
			svc := &serviceImpl{
				roleStore: mockRoleStore,
			}
			mockRoleStore.EXPECT().
				UpsertAccessScope(gomock.Any(), protomock.GoMockMatcherEqualMessage(tc.expectedAccessScope)).
				Times(1).
				Return(tc.expectedStoreError)

			asID, err := svc.createAccessScope(ctx, tc.input)

			if tc.expectedStoreError != nil {
				assert.Empty(it, asID)
				assert.ErrorIs(it, err, tc.expectedStoreError)
			} else {
				assert.Equal(it, tc.expectedAccessScope.GetId(), asID)
				assert.NoError(it, err)
			}
		})
	}
}

func computeAccessScopeID(targetScopes []*central.RequestedRoleClusterScope) string {
	clusterScopes := make([]string, 0, len(targetScopes))
	for _, targetScope := range targetScopes {
		var namespaceScope string
		if targetScope.GetFullClusterAccess() {
			namespaceScope = clusterWildCard
		} else {
			namespaceScope = strings.Join(targetScope.GetNamespaces(), secondaryListSeparator)
		}
		clusterScopes = append(
			clusterScopes,
			fmt.Sprintf("%s%s%s", targetScope.GetClusterName(), keyValueSeparator, namespaceScope),
		)
	}
	accessScopeString := strings.Join(clusterScopes, primaryListSeparator)
	return declarativeconfig.NewDeclarativeAccessScopeUUID(accessScopeString).String()
}

func testAccessScope(targetScopes []*central.RequestedRoleClusterScope) *storage.SimpleAccessScope {
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
		if targetScope.GetFullClusterAccess() {
			accessScope.Rules.IncludedClusters = append(
				accessScope.Rules.IncludedClusters,
				targetScope.GetClusterName(),
			)
		} else {
			for _, namespace := range targetScope.GetNamespaces() {
				accessScope.Rules.IncludedNamespaces = append(
					accessScope.Rules.IncludedNamespaces,
					&storage.SimpleAccessScope_Rules_Namespace{
						ClusterName:   targetScope.GetClusterName(),
						NamespaceName: namespace,
					},
				)
			}
		}
	}
	return accessScope
}

func TestCreateRole(t *testing.T) {
	deploymentPermission := []string{"Deployment"}
	targetCluster1 := "cluster 1"
	targetCluster2 := "cluster 2"
	targetNamespaceA := "namespace A"
	requestFullCluster := &central.RequestedRoleClusterScope{
		ClusterName:       targetCluster1,
		FullClusterAccess: true,
	}
	requestSingleNamespace := &central.RequestedRoleClusterScope{
		ClusterName:       targetCluster2,
		FullClusterAccess: false,
		Namespaces:        []string{targetNamespaceA},
	}
	for name, tc := range map[string]struct {
		input                  *central.GenerateTokenForPermissionsAndScopeRequest
		expectedPermissionSet  *storage.PermissionSet
		expectedAccessScope    *storage.SimpleAccessScope
		expectedRole           *storage.Role
		expectedRoleStoreError error
	}{
		"nil input, successful storage (role with no permission and empty scope)": {
			input:                  nil,
			expectedPermissionSet:  testPermissionSet(make([]string, 0)),
			expectedAccessScope:    testAccessScope(make([]*central.RequestedRoleClusterScope, 0)),
			expectedRole:           testRole(make([]string, 0), make([]*central.RequestedRoleClusterScope, 0)),
			expectedRoleStoreError: nil,
		},
		"nil input, failed storage (role with no permission and empty scope)": {
			input:                  nil,
			expectedPermissionSet:  testPermissionSet(make([]string, 0)),
			expectedAccessScope:    testAccessScope(make([]*central.RequestedRoleClusterScope, 0)),
			expectedRole:           testRole(make([]string, 0), make([]*central.RequestedRoleClusterScope, 0)),
			expectedRoleStoreError: errDummy,
		},
		"request for single full cluster access to deployments, successful storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ReadPermissions: deploymentPermission,
				ClusterScopes:   []*central.RequestedRoleClusterScope{requestFullCluster},
			},
			expectedPermissionSet:  testPermissionSet(deploymentPermission),
			expectedAccessScope:    testAccessScope([]*central.RequestedRoleClusterScope{requestFullCluster}),
			expectedRole:           testRole(deploymentPermission, []*central.RequestedRoleClusterScope{requestFullCluster}),
			expectedRoleStoreError: nil,
		},
		"request for single full cluster access to deployments, failed storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ReadPermissions: deploymentPermission,
				ClusterScopes:   []*central.RequestedRoleClusterScope{requestFullCluster},
			},
			expectedPermissionSet:  testPermissionSet(deploymentPermission),
			expectedAccessScope:    testAccessScope([]*central.RequestedRoleClusterScope{requestFullCluster}),
			expectedRole:           testRole(deploymentPermission, []*central.RequestedRoleClusterScope{requestFullCluster}),
			expectedRoleStoreError: errDummy,
		},
		"request for single namespace access to deployments, successful storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ReadPermissions: deploymentPermission,
				ClusterScopes:   []*central.RequestedRoleClusterScope{requestSingleNamespace},
			},
			expectedPermissionSet:  testPermissionSet(deploymentPermission),
			expectedAccessScope:    testAccessScope([]*central.RequestedRoleClusterScope{requestSingleNamespace}),
			expectedRole:           testRole(deploymentPermission, []*central.RequestedRoleClusterScope{requestSingleNamespace}),
			expectedRoleStoreError: nil,
		},
		"request for single namespace access to deployments, failed storage": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ReadPermissions: deploymentPermission,
				ClusterScopes:   []*central.RequestedRoleClusterScope{requestSingleNamespace},
			},
			expectedPermissionSet:  testPermissionSet(deploymentPermission),
			expectedAccessScope:    testAccessScope([]*central.RequestedRoleClusterScope{requestSingleNamespace}),
			expectedRole:           testRole(deploymentPermission, []*central.RequestedRoleClusterScope{requestSingleNamespace}),
			expectedRoleStoreError: errDummy,
		},
	} {
		t.Run(name, func(it *testing.T) {
			ctx := sac.WithAllAccess(it.Context())
			mockCtrl := gomock.NewController(it)
			mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
			svc := &serviceImpl{
				roleStore: mockRoleStore,
			}
			setNormalRoleStoreExpectations(
				tc.expectedPermissionSet,
				tc.expectedAccessScope,
				tc.expectedRole,
				tc.expectedRoleStoreError,
				mockRoleStore,
			)

			roleName, err := svc.createRole(ctx, tc.input)

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
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		svc := &serviceImpl{
			roleStore: mockRoleStore,
		}
		expectedPermissionSet := testPermissionSet(deploymentPermission)
		expectedAccessScope := testAccessScope([]*central.RequestedRoleClusterScope{requestSingleNamespace})
		accessScopeCreationErr := errors.New("access scope creation error")
		mockRoleStore.EXPECT().
			UpsertPermissionSet(gomock.Any(), protomock.GoMockMatcherEqualMessage(expectedPermissionSet)).
			Times(1).
			Return(nil)
		mockRoleStore.EXPECT().
			UpsertAccessScope(gomock.Any(), protomock.GoMockMatcherEqualMessage(expectedAccessScope)).
			Times(1).
			Return(accessScopeCreationErr)

		input := &central.GenerateTokenForPermissionsAndScopeRequest{
			ReadPermissions: deploymentPermission,
			ClusterScopes:   []*central.RequestedRoleClusterScope{requestSingleNamespace},
		}

		roleName, err := svc.createRole(ctx, input)

		assert.Empty(it, roleName)
		assert.ErrorIs(it, err, accessScopeCreationErr)
	})

	t.Run("permission set creation failure", func(it *testing.T) {
		ctx := sac.WithAllAccess(it.Context())
		mockCtrl := gomock.NewController(it)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		svc := &serviceImpl{
			roleStore: mockRoleStore,
		}
		expectedPermissionSet := testPermissionSet(deploymentPermission)
		permissionSetCreationErr := errors.New("permission set creation error")
		mockRoleStore.EXPECT().
			UpsertPermissionSet(gomock.Any(), protomock.GoMockMatcherEqualMessage(expectedPermissionSet)).
			Times(1).
			Return(permissionSetCreationErr)

		input := &central.GenerateTokenForPermissionsAndScopeRequest{
			ReadPermissions: deploymentPermission,
			ClusterScopes:   []*central.RequestedRoleClusterScope{requestSingleNamespace},
		}

		roleName, err := svc.createRole(ctx, input)

		assert.Empty(it, roleName)
		assert.ErrorIs(it, err, permissionSetCreationErr)
	})
}

func computeRoleName(readResources []string, targetScopes []*central.RequestedRoleClusterScope) string {
	permissionSetID := computePermissionSetID(readResources)
	accessScopeID := computeAccessScopeID(targetScopes)
	return fmt.Sprintf(roleNameFormat, permissionSetID, accessScopeID)
}

func testRole(readResources []string, targetScopes []*central.RequestedRoleClusterScope) *storage.Role {
	permissionSetID := computePermissionSetID(readResources)
	accessScopeID := computeAccessScopeID(targetScopes)
	role := &storage.Role{
		Name:            computeRoleName(readResources, targetScopes),
		Description:     "Generated role for OCP console plugin",
		PermissionSetId: permissionSetID,
		AccessScopeId:   accessScopeID,
		Traits:          generatedObjectTraits.CloneVT(),
	}
	return role
}

func TestGetExpiresAt(t *testing.T) {
	for name, tc := range map[string]struct {
		input              *central.GenerateTokenForPermissionsAndScopeRequest
		expectsErr         bool
		expectedExpiration time.Time
	}{
		"nil input": {
			input:      nil,
			expectsErr: true,
		},
		"input without requested validity": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ValidFor: nil,
			},
			expectsErr: true,
		},
		"input with invalid requested validity": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ValidFor: &durationpb.Duration{
					Seconds: 60,
					Nanos:   -654321987,
				},
			},
			expectsErr: true,
		},
		"input with negative requested validity": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ValidFor: &durationpb.Duration{
					Seconds: -60,
					Nanos:   -654321987,
				},
			},
			expectsErr: true,
		},
		"valid input": {
			input: &central.GenerateTokenForPermissionsAndScopeRequest{
				ValidFor: &durationpb.Duration{
					Seconds: 300,
				},
			},
			expectsErr:         false,
			expectedExpiration: expectedExpiration,
		},
	} {
		t.Run(name, func(it *testing.T) {
			svc := &serviceImpl{now: testClock}
			expiresAt, err := svc.getExpiresAt(it.Context(), tc.input)
			if tc.expectsErr {
				assert.Error(it, err)
				assert.Zero(it, expiresAt)
			} else {
				assert.NoError(it, err)
				assert.Equal(it, tc.expectedExpiration, expiresAt)
			}
		})
	}
}

func TestGenerateTokenForPermissionsAndScope(t *testing.T) {
	deploymentPermission := []string{"Deployment"}
	requestSingleNamespace := &central.RequestedRoleClusterScope{
		ClusterName:       "cluster 1",
		FullClusterAccess: false,
		Namespaces:        []string{"namespace A"},
	}
	deploymentPS := testPermissionSet(deploymentPermission)
	singleNSScope := testAccessScope(
		[]*central.RequestedRoleClusterScope{requestSingleNamespace},
	)
	expectedRole := testRole(
		deploymentPermission,
		[]*central.RequestedRoleClusterScope{requestSingleNamespace},
	)

	createService := func(issuer tokens.Issuer, roleStore roleDataStore.DataStore) *serviceImpl {
		return &serviceImpl{
			issuer:    issuer,
			roleStore: roleStore,
			now:       testClock,
		}
	}

	t.Run("no requested validity", func(it *testing.T) {
		input := &central.GenerateTokenForPermissionsAndScopeRequest{
			ReadPermissions: deploymentPermission,
			ClusterScopes:   []*central.RequestedRoleClusterScope{requestSingleNamespace},
			ValidFor:        nil,
		}

		mockCtrl := gomock.NewController(it)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		svc := createService(nil, mockRoleStore)
		setNormalRoleStoreExpectations(deploymentPS, singleNSScope, expectedRole, nil, mockRoleStore)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(t.Context(), input)
		assert.Nil(it, rsp)
		assert.Error(it, err)
	})
	t.Run("failed role creation", func(it *testing.T) {
		input := &central.GenerateTokenForPermissionsAndScopeRequest{
			ReadPermissions: deploymentPermission,
			ClusterScopes:   []*central.RequestedRoleClusterScope{requestSingleNamespace},
			ValidFor:        nil,
		}

		mockCtrl := gomock.NewController(it)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		svc := createService(nil, mockRoleStore)
		mockRoleStore.EXPECT().
			UpsertPermissionSet(
				gomock.Any(),
				protomock.GoMockMatcherEqualMessage(deploymentPS),
			).Times(1).Return(errDummy)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(t.Context(), input)
		assert.Nil(it, rsp)
		assert.Error(it, err)
	})
	t.Run("token issuer failure", func(it *testing.T) {
		input := &central.GenerateTokenForPermissionsAndScopeRequest{
			ReadPermissions: deploymentPermission,
			ClusterScopes:   []*central.RequestedRoleClusterScope{requestSingleNamespace},
			ValidFor:        &durationpb.Duration{Seconds: 300},
		}

		mockCtrl := gomock.NewController(it)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		mockIssuer := tokensMocks.NewMockIssuer(mockCtrl)
		svc := createService(mockIssuer, mockRoleStore)
		setNormalRoleStoreExpectations(deploymentPS, singleNSScope, expectedRole, nil, mockRoleStore)
		expectedClaims := tokens.RoxClaims{
			RoleNames: []string{expectedRole.GetName()},
			Name: fmt.Sprintf(
				"Generated claims for role %s expiring at %s",
				expectedRole.GetName(),
				expectedExpiration.Format(time.RFC3339Nano),
			),
			ExpireAt: &expectedExpiration,
		}
		mockIssuer.EXPECT().
			Issue(gomock.Any(), expectedClaims).
			Times(1).Return(nil, errDummy)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(t.Context(), input)
		assert.Nil(it, rsp)
		assert.Error(it, err)
	})
	t.Run("success", func(it *testing.T) {
		input := &central.GenerateTokenForPermissionsAndScopeRequest{
			ReadPermissions: deploymentPermission,
			ClusterScopes:   []*central.RequestedRoleClusterScope{requestSingleNamespace},
			ValidFor:        &durationpb.Duration{Seconds: 300},
		}

		mockCtrl := gomock.NewController(it)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		mockIssuer := tokensMocks.NewMockIssuer(mockCtrl)
		svc := createService(mockIssuer, mockRoleStore)
		setNormalRoleStoreExpectations(deploymentPS, singleNSScope, expectedRole, nil, mockRoleStore)
		expectedClaims := tokens.RoxClaims{
			RoleNames: []string{expectedRole.GetName()},
			Name: fmt.Sprintf(
				"Generated claims for role %s expiring at %s",
				expectedRole.GetName(),
				expectedExpiration.Format(time.RFC3339Nano),
			),
			ExpireAt: &expectedExpiration,
		}
		mockIssuer.EXPECT().
			Issue(gomock.Any(), expectedClaims).
			Times(1).Return(&tokens.TokenInfo{Token: "the quick brown fox jumps over the lazy dog"}, nil)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(t.Context(), input)
		assert.NotNil(it, rsp)
		protoassert.Equal(
			it,
			&central.GenerateTokenForPermissionsAndScopeResponse{
				Token: "the quick brown fox jumps over the lazy dog",
			},
			rsp,
		)
		assert.NoError(it, err)
	})
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
