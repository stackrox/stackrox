package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	clusterDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	roleDataStoreMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/internalapi/central/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	tokensMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/authn"
	authnMocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/durationpb"
)

const testSensorClusterID = "cluster 1"

var (
	errDummy = errors.New("test error")

	// permissivePolicy allows the permissions used in existing tests.
	permissivePolicy = newTokenPolicy(1*time.Hour, map[string]v1.Access{
		"Deployment": v1.Access_READ_ACCESS,
		"Image":      v1.Access_READ_ACCESS,
	})
)

// sensorContext returns a context with a mock sensor identity injected.
func sensorContext(t testing.TB, ctrl *gomock.Controller, clusterID string) context.Context {
	mockIdentity := authnMocks.NewMockIdentity(ctrl)
	mockIdentity.EXPECT().Service().Return(&storage.ServiceIdentity{
		Id:   clusterID,
		Type: storage.ServiceType_SENSOR_SERVICE,
	}).AnyTimes()
	return authn.ContextWithIdentity(t.Context(), mockIdentity, t)
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
		"valid input": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Lifetime: &durationpb.Duration{
					Seconds: 300,
				},
			},
			expectsErr:         false,
			expectedExpiration: testTokenExpiry,
		},
	} {
		t.Run(name, func(it *testing.T) {
			svc := &serviceImpl{now: testClock, policy: permissivePolicy}
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
		ClusterId:         fixtureconsts.Cluster1,
		FullClusterAccess: false,
		Namespaces:        []string{"namespace A"},
	}
	requestFullCluster := &v1.ClusterScope{
		ClusterId:         fixtureconsts.Cluster2,
		FullClusterAccess: true,
	}

	createService := func(
		t testing.TB,
		issuer tokens.Issuer,
		clusterStore clusterDataStore.DataStore,
		roleStore roleDataStore.DataStore,
		policy *tokenPolicy,
	) *serviceImpl {
		t.Helper()
		return &serviceImpl{
			issuer: issuer,
			roleManager: &roleManager{
				clusterStore: clusterStore,
				roleStore:    roleStore,
			},
			now:    testClock,
			policy: policy,
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
		svc := createService(it, nil, mockClusterStore, mockRoleStore, permissivePolicy)
		ctx := sensorContext(it, mockCtrl, testSensorClusterID)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(ctx, input)
		assert.Nil(it, rsp)
		assert.Error(it, err)
	})
	t.Run("failed role creation", func(it *testing.T) {
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   deploymentPermission,
			ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			Lifetime:      testExpirationDuration,
		}

		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		svc := createService(it, nil, mockClusterStore, mockRoleStore, permissivePolicy)
		mockClusterStore.EXPECT().
			GetClusterName(gomock.Any(), fixtureconsts.Cluster1).
			Times(1).
			Return("", false, errDummy)
		ctx := sensorContext(it, mockCtrl, fixtureconsts.Cluster1)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(ctx, input)
		assert.Nil(it, rsp)
		assert.Error(it, err)
	})
	t.Run("token issuer failure", func(it *testing.T) {
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   deploymentPermission,
			ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			Lifetime:      testExpirationDuration,
		}

		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		mockIssuer := tokensMocks.NewMockIssuer(mockCtrl)
		svc := createService(it, mockIssuer, mockClusterStore, mockRoleStore, permissivePolicy)
		setClusterStoreExpectations(input, mockClusterStore)
		expectedClaims := tokens.RoxClaims{
			InternalRoles: []*tokens.InternalRole{
				{
					RoleName:    internalRoleName,
					Permissions: map[string]string{"Deployment": storage.Access_READ_ACCESS.String()},
					ClusterScopes: []*tokens.ClusterScope{
						{
							ClusterName: fixtureconsts.Cluster1,
							Namespaces:  []string{"namespace A"},
						},
					},
				},
			},
			Name: fmt.Sprintf(
				claimNameFormat,
				"internal role",
				testTokenExpiry.Format(time.RFC3339Nano),
			),
		}
		mockIssuer.EXPECT().
			Issue(gomock.Any(), expectedClaims, gomock.Any()).
			Times(1).Return(nil, errDummy)
		ctx := sensorContext(it, mockCtrl, fixtureconsts.Cluster1)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(ctx, input)
		assert.Nil(it, rsp)
		assert.Error(it, err)
	})
	for name, tc := range map[string]struct {
		input          *v1.GenerateTokenForPermissionsAndScopeRequest
		expectedClaims tokens.RoxClaims
		tokenString    string
	}{
		"success - standard request": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   deploymentPermission,
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
				Lifetime:      testExpirationDuration,
			},
			expectedClaims: tokens.RoxClaims{
				InternalRoles: []*tokens.InternalRole{
					{
						RoleName:    internalRoleName,
						Permissions: map[string]string{"Deployment": storage.Access_READ_ACCESS.String()},
						ClusterScopes: []*tokens.ClusterScope{
							{
								ClusterName: fixtureconsts.Cluster1,
								Namespaces:  []string{"namespace A"},
							},
						},
					},
				},
				Name: fmt.Sprintf(
					"Generated claims for role %s expiring at %s",
					"internal role",
					testTokenExpiry.Format(time.RFC3339Nano),
				),
			},
			tokenString: "the quick brown fox jumps over the lazy dog",
		},
		"success - no requested permission": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
				Lifetime:      testExpirationDuration,
			},
			expectedClaims: tokens.RoxClaims{
				InternalRoles: []*tokens.InternalRole{
					{
						RoleName:    internalRoleName,
						Permissions: make(map[string]string),
						ClusterScopes: []*tokens.ClusterScope{
							{
								ClusterName: fixtureconsts.Cluster1,
								Namespaces:  []string{"namespace A"},
							},
						},
					},
				},
				Name: fmt.Sprintf(
					"Generated claims for role %s expiring at %s",
					"internal role",
					testTokenExpiry.Format(time.RFC3339Nano),
				),
			},
			tokenString: "In the days when everybody started fair, the Leopard lived in a place called the High Veldt.",
		},
		"success - no requested scope": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions: deploymentPermission,
				Lifetime:    testExpirationDuration,
			},
			expectedClaims: tokens.RoxClaims{
				InternalRoles: []*tokens.InternalRole{
					{
						RoleName:    internalRoleName,
						Permissions: map[string]string{"Deployment": storage.Access_READ_ACCESS.String()},
					},
				},
				Name: fmt.Sprintf(
					"Generated claims for role %s expiring at %s",
					"internal role",
					testTokenExpiry.Format(time.RFC3339Nano),
				),
			},
			tokenString: "In the high and far-off times the elephant had no trunk.",
		},
		"success - multiple permissions and cluster scopes": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions: map[string]v1.Access{
					"Deployment": v1.Access_READ_ACCESS,
					"Image":      v1.Access_READ_WRITE_ACCESS,
				},
				ClusterScopes: []*v1.ClusterScope{
					requestSingleNamespace,
					requestFullCluster,
				},
				Lifetime: testExpirationDuration,
			},
			expectedClaims: tokens.RoxClaims{
				InternalRoles: []*tokens.InternalRole{
					{
						RoleName: internalRoleName,
						Permissions: map[string]string{
							"Deployment": storage.Access_READ_ACCESS.String(),
							"Image":      storage.Access_READ_WRITE_ACCESS.String(),
						},
						ClusterScopes: []*tokens.ClusterScope{
							{
								ClusterName: fixtureconsts.Cluster1,
								Namespaces:  []string{"namespace A"},
							},
							{
								ClusterName:       fixtureconsts.Cluster2,
								ClusterFullAccess: true,
							},
						},
					},
				},
				Name: fmt.Sprintf(
					"Generated claims for role %s expiring at %s",
					"internal role",
					testTokenExpiry.Format(time.RFC3339Nano),
				),
			},
			tokenString: "Hear and attend and listen; for this befell and happened and became and was, when the Tame animals were wild.",
		},
	} {
		t.Run(name, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
			mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
			mockIssuer := tokensMocks.NewMockIssuer(mockCtrl)
			svc := createService(mockIssuer, mockClusterStore, mockRoleStore)
			setClusterStoreExpectations(tc.input, mockClusterStore)

			mockIssuer.EXPECT().
				Issue(gomock.Any(), tc.expectedClaims, gomock.Any()).
				Times(1).Return(&tokens.TokenInfo{Token: tc.tokenString}, nil)

			rsp, err := svc.GenerateTokenForPermissionsAndScope(t.Context(), tc.input)
			assert.NotNil(it, rsp)
			protoassert.Equal(
				it,
				&v1.GenerateTokenForPermissionsAndScopeResponse{Token: tc.tokenString},
				rsp,
			)
			assert.NoError(it, err)
		})
	}
	t.Run("success - standard request", func(it *testing.T) {
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   deploymentPermission,
			ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			Lifetime:      testExpirationDuration,
		}

		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		mockIssuer := tokensMocks.NewMockIssuer(mockCtrl)
		svc := createService(it, mockIssuer, mockClusterStore, mockRoleStore, permissivePolicy)
		setClusterStoreExpectations(input, mockClusterStore)
		expectedClaims := tokens.RoxClaims{
			InternalRoles: []*tokens.InternalRole{
				{
					RoleName:    internalRoleName,
					Permissions: map[string]string{"Deployment": storage.Access_READ_ACCESS.String()},
					ClusterScopes: []*tokens.ClusterScope{
						{
							ClusterName: fixtureconsts.Cluster1,
							Namespaces:  []string{"namespace A"},
						},
					},
				},
			},
			Name: fmt.Sprintf(
				"Generated claims for role %s expiring at %s",
				"internal role",
				testTokenExpiry.Format(time.RFC3339Nano),
			),
		}
		mockIssuer.EXPECT().
			Issue(gomock.Any(), expectedClaims, gomock.Any()).
			Times(1).Return(&tokens.TokenInfo{Token: "the quick brown fox jumps over the lazy dog"}, nil)
		ctx := sensorContext(it, mockCtrl, fixtureconsts.Cluster1)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(ctx, input)
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
	t.Run("permission not in allowlist rejects request", func(it *testing.T) {
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions: map[string]v1.Access{
				"NetworkGraph": v1.Access_READ_ACCESS,
			},
			ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			Lifetime:      testExpirationDuration,
		}
		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		svc := createService(it, nil, mockClusterStore, mockRoleStore, permissivePolicy)
		ctx := sensorContext(it, mockCtrl, fixtureconsts.Cluster1)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(ctx, input)
		assert.Nil(it, rsp)
		assert.ErrorIs(it, err, errox.InvalidArgs)
	})
	t.Run("access level exceeds allowlist rejects request", func(it *testing.T) {
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions: map[string]v1.Access{
				"Deployment": v1.Access_READ_WRITE_ACCESS,
			},
			ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			Lifetime:      testExpirationDuration,
		}
		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		svc := createService(it, nil, mockClusterStore, mockRoleStore, permissivePolicy)
		ctx := sensorContext(it, mockCtrl, fixtureconsts.Cluster1)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(ctx, input)
		assert.Nil(it, rsp)
		assert.ErrorIs(it, err, errox.InvalidArgs)
	})
	t.Run("cluster scope mismatch rejects request", func(it *testing.T) {
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions: deploymentPermission,
			ClusterScopes: []*v1.ClusterScope{
				{ClusterId: "other-cluster", Namespaces: []string{"ns"}},
			},
			Lifetime: testExpirationDuration,
		}
		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		svc := createService(it, nil, mockClusterStore, mockRoleStore, permissivePolicy)
		ctx := sensorContext(it, mockCtrl, fixtureconsts.Cluster1)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(ctx, input)
		assert.Nil(it, rsp)
		assert.ErrorIs(it, err, errox.InvalidArgs)
	})
	t.Run("lifetime capping", func(it *testing.T) {
		shortMaxPolicy := newTokenPolicy(10*time.Second, map[string]v1.Access{
			"Deployment": v1.Access_READ_ACCESS,
		})
		input := &v1.GenerateTokenForPermissionsAndScopeRequest{
			Permissions:   deploymentPermission,
			ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
			// 300 seconds requested, but policy caps at 10 seconds.
			Lifetime: testExpirationDuration,
		}

		cappedExpiry := testClock().Add(10 * time.Second)

		mockCtrl := gomock.NewController(it)
		mockClusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
		mockRoleStore := roleDataStoreMocks.NewMockDataStore(mockCtrl)
		mockIssuer := tokensMocks.NewMockIssuer(mockCtrl)
		svc := createService(it, mockIssuer, mockClusterStore, mockRoleStore, shortMaxPolicy)
		setClusterStoreExpectations(input, mockClusterStore)
		expectedClaims := tokens.RoxClaims{
			Name: fmt.Sprintf(
				claimNameFormat,
				"internal role",
				cappedExpiry.Format(time.RFC3339Nano),
			),
			InternalRoles: []*tokens.InternalRole{
				{
					RoleName: internalRoleName,
					Permissions: map[string]string{
						"Deployment": storage.Access_READ_ACCESS.String(),
					},
					ClusterScopes: []*tokens.ClusterScope{
						{
							ClusterName: fixtureconsts.Cluster1,
							Namespaces:  []string{"namespace A"},
						},
					},
				},
			},
		}
		mockIssuer.EXPECT().
			Issue(gomock.Any(), expectedClaims, gomock.Any()).
			Times(1).Return(&tokens.TokenInfo{Token: "capped-token"}, nil)
		ctx := sensorContext(it, mockCtrl, fixtureconsts.Cluster1)

		rsp, err := svc.GenerateTokenForPermissionsAndScope(ctx, input)
		assert.NotNil(it, rsp)
		assert.NoError(it, err)
		assert.Equal(it, "capped-token", rsp.GetToken())
	})
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
