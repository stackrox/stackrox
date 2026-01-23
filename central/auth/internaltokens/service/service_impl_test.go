package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	roleDataStoreMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/pkg/auth/tokens"
	tokensMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protomock"
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

func TestGetExpiresAt(t *testing.T) {
	for name, tc := range map[string]struct {
		input              *v1.GenerateTokenForPermissionsAndScopeRequest
		expectsErr         bool
		expectedExpiration time.Time
	}{
		"nil input": {
			input:      nil,
			expectsErr: true,
		},
		"input without requested validity": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Lifetime: nil,
			},
			expectsErr: true,
		},
		"input with invalid requested validity": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Lifetime: &durationpb.Duration{
					Seconds: 60,
					Nanos:   -654321987,
				},
			},
			expectsErr: true,
		},
		"input with negative requested validity": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Lifetime: &durationpb.Duration{
					Seconds: -60,
					Nanos:   -654321987,
				},
			},
			expectsErr: true,
		},
		"valid input": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Lifetime: &durationpb.Duration{
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
				fmt.Println(err.Error())
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
	deploymentPermission := map[string]v1.Access{
		"Deployment": v1.Access_READ_ACCESS,
	}
	requestSingleNamespace := &v1.ClusterScope{
		ClusterId:         "cluster 1",
		FullClusterAccess: false,
		Namespaces:        []string{"namespace A"},
	}
	deploymentPS := testPermissionSet(deploymentPermission)
	singleNSScope := testAccessScope(
		[]*v1.ClusterScope{requestSingleNamespace},
	)
	expectedRole := testRole(
		deploymentPermission,
		[]*v1.ClusterScope{requestSingleNamespace},
	)

	createService := func(
		issuer tokens.Issuer,
		clusterStore clusterDataStore.DataStore,
		roleStore roleDataStore.DataStore,
	) *serviceImpl {
		return &serviceImpl{
			issuer: issuer,
			roleManager: &roleManager{
				clusterStore: clusterStore,
				roleStore:    roleStore,
			},
			now: testClock,
		}
	}

	t.Run("no requested validity", func(it *testing.T) {
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   deploymentPermission,
			ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			Lifetime:      nil,
		}

		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		svc := createService(nil, mockClusterStore, mockRoleStore)
		// Note: With the new code structure, getExpiresAt is called first.
		// Since Lifetime is nil, getExpiresAt returns an error before createRole is called,
		// so we don't set up any role store expectations.

		rsp, err := svc.GenerateTokenForPermissionsAndScope(t.Context(), input)
		assert.Nil(it, rsp)
		assert.Error(it, err)
	})
	t.Run("failed role creation", func(it *testing.T) {
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   deploymentPermission,
			ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			Lifetime:      &durationpb.Duration{Seconds: 300},
		}

		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		svc := createService(nil, mockClusterStore, mockRoleStore)
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
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   deploymentPermission,
			ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			Lifetime:      &durationpb.Duration{Seconds: 300},
		}

		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		mockIssuer := tokensMocks.NewMockIssuer(mockCtrl)
		svc := createService(mockIssuer, mockClusterStore, mockRoleStore)
		setClusterStoreExpectations(input, mockClusterStore)
		setNormalRoleStoreExpectations(deploymentPS, singleNSScope, expectedRole, nil, mockRoleStore)
		expectedClaims := tokens.RoxClaims{
			RoleNames: []string{expectedRole.GetName()},
			Name: fmt.Sprintf(
				claimNameFormat,
				expectedRole.GetName(),
				expectedExpiration.Format(time.RFC3339Nano),
			),
		}
		mockIssuer.EXPECT().
			Issue(gomock.Any(), expectedClaims, gomock.Any()).
			Times(1).Return(nil, errDummy)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(t.Context(), input)
		assert.Nil(it, rsp)
		assert.Error(it, err)
	})
	t.Run("success", func(it *testing.T) {
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   deploymentPermission,
			ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			Lifetime:      &durationpb.Duration{Seconds: 300},
		}

		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		mockIssuer := tokensMocks.NewMockIssuer(mockCtrl)
		svc := createService(mockIssuer, mockClusterStore, mockRoleStore)
		setClusterStoreExpectations(input, mockClusterStore)
		setNormalRoleStoreExpectations(deploymentPS, singleNSScope, expectedRole, nil, mockRoleStore)
		expectedClaims := tokens.RoxClaims{
			RoleNames: []string{expectedRole.GetName()},
			Name: fmt.Sprintf(
				"Generated claims for role %s expiring at %s",
				expectedRole.GetName(),
				expectedExpiration.Format(time.RFC3339Nano),
			),
		}
		mockIssuer.EXPECT().
			Issue(gomock.Any(), expectedClaims, gomock.Any()).
			Times(1).Return(&tokens.TokenInfo{Token: "the quick brown fox jumps over the lazy dog"}, nil)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(t.Context(), input)
		assert.NotNil(it, rsp)
		protoassert.Equal(
			it,
			&v1.GenerateTokenForPermissionsAndScopeResponse{
				Token: "the quick brown fox jumps over the lazy dog",
			},
			rsp,
		)
		assert.NoError(it, err)
	})
}
