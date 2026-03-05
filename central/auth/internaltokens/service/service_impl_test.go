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

const (
	testSensorClusterID = "cluster 1"

	deploymentResource   = "Deployment"
	imageResource        = "Image"
	networkGraphResource = "NetworkGraph"
)

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

type mockContainer struct {
	clusterStore *clusterDataStoreMocks.MockDataStore
	roleStore    *roleDataStoreMocks.MockDataStore
	issuer       *tokensMocks.MockIssuer
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
		ClusterId:         fixtureconsts.Cluster1,
		FullClusterAccess: true,
	}

	clusterIDNameMap := map[string]string{fixtureconsts.Cluster1: testSensorClusterID}

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

	// error cases
	for name, tc := range map[string]struct {
		input *v1.GenerateTokenForPermissionsAndScopeRequest
		setup func(*mockContainer)
		// policy      *tokenPolicy
		expectedErr error
	}{
		"nil request": {
			// The request processing fails at the step where
			// the token expiration is computed (missing input).
			input:       nil,
			expectedErr: errox.InvalidArgs,
		},
		"no requested validity": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   deploymentPermission,
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
				// The missing lifetime field causes computation of the
				// token expiration to fail.
				Lifetime: nil,
			},
			expectedErr: errox.InvalidArgs,
		},
		"failed role creation": {
			// The input is valid, the error returned by the (mocked) role store
			// is propagated.
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   deploymentPermission,
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
				Lifetime:      testExpirationDuration,
			},
			setup: func(mocks *mockContainer) {
				mocks.clusterStore.EXPECT().
					GetClusterName(gomock.Any(), fixtureconsts.Cluster1).
					Times(1).
					Return("", false, errDummy)
			},
			expectedErr: errDummy,
		},
		"token issuer failure": {
			// The input is valid. The mock setup lets the flow succeed
			// up to the point where the (mocked) token issuer is called.
			// The expected claims are the outcome of the process so far,
			// the issuer error is propagated.
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   deploymentPermission,
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
				Lifetime:      testExpirationDuration,
			},
			setup: func(mocks *mockContainer) {
				setClusterStoreExpectations(mocks.clusterStore, clusterIDNameMap)
				expectedClaims := tokens.RoxClaims{
					InternalRoles: []*tokens.InternalRole{
						{
							RoleName:      internalRoleName,
							ReadResources: []string{"Deployment"},
							ClusterScopes: []*tokens.ClusterScope{
								{
									ClusterName: testSensorClusterID,
									Namespaces:  []string{"namespace A"},
								},
							},
							/*
								Clusters: map[string][]string{
									testSensorClusterID: {"namespace A"},
								},
							*/
						},
					},
					Name: fmt.Sprintf(
						claimNameFormat,
						internalRoleName,
						testTokenExpiry.Format(time.RFC3339Nano),
					),
				}
				mocks.issuer.EXPECT().
					Issue(gomock.Any(), expectedClaims, gomock.Any()).
					Times(1).Return(nil, errDummy)
			},
			expectedErr: errDummy,
		},
		"permission not in allowlist rejects request": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions: map[string]v1.Access{
					// The tokenPolicy used for these error tests (permissivePolicy)
					// does not allow actions on the NetworkGraph resource.
					"NetworkGraph": v1.Access_READ_ACCESS,
				},
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
				Lifetime:      testExpirationDuration,
			},
			expectedErr: errox.InvalidArgs,
		},
		"access level exceeds allowlist rejects request": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions: map[string]v1.Access{
					// The tokenPolicy used for these error tests (permissivePolicy)
					// only allows read actions on the Deployment resource.
					"Deployment": v1.Access_READ_WRITE_ACCESS,
				},
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
				Lifetime:      testExpirationDuration,
			},
			expectedErr: errox.InvalidArgs,
		},
		"cluster scope mismatch rejects request": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions: deploymentPermission,
				ClusterScopes: []*v1.ClusterScope{
					// The context in use for these error tests identifies
					// the source cluster as "cluster 1" (testSensorClusterID).
					// The requested scope for "other-cluster" does not match.
					{ClusterId: "other-cluster", Namespaces: []string{"ns"}},
				},
				Lifetime: testExpirationDuration,
			},
			expectedErr: errox.InvalidArgs,
		},
	} {
		t.Run(name, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			mocks := &mockContainer{
				clusterStore: clusterDataStoreMocks.NewMockDataStore(mockCtrl),
				roleStore:    roleDataStoreMocks.NewMockDataStore(mockCtrl),
				issuer:       tokensMocks.NewMockIssuer(mockCtrl),
			}
			if tc.setup != nil {
				tc.setup(mocks)
			}
			svc := createService(it, mocks.issuer, mocks.clusterStore, mocks.roleStore, permissivePolicy)
			ctx := sensorContext(it, mockCtrl, fixtureconsts.Cluster1)

			rsp, err := svc.GenerateTokenForPermissionsAndScope(ctx, tc.input)
			assert.Nil(it, rsp)
			assert.ErrorIs(it, err, tc.expectedErr)
		})
	}

	// success cases
	for name, tc := range map[string]struct {
		input       *v1.GenerateTokenForPermissionsAndScopeRequest
		policy      *tokenPolicy
		setup       func(*testing.T, *mockContainer)
		expectedRsp *v1.GenerateTokenForPermissionsAndScopeResponse
	}{
		"success - standard request": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   deploymentPermission,
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
				Lifetime:      testExpirationDuration,
			},
			policy: permissivePolicy,
			setup: func(_ *testing.T, mocks *mockContainer) {
				setClusterStoreExpectations(mocks.clusterStore, clusterIDNameMap)
				expectedClaims := tokens.RoxClaims{
					InternalRoles: []*tokens.InternalRole{
						{
							RoleName:      internalRoleName,
							ReadResources: []string{deploymentResource},
							ClusterScopes: []*tokens.ClusterScope{
								{
									ClusterName: testSensorClusterID,
									Namespaces:  []string{"namespace A"},
								},
							},
						},
					},
					Name: fmt.Sprintf(
						"Generated claims for role %s expiring at %s",
						internalRoleName,
						testTokenExpiry.Format(time.RFC3339Nano),
					),
				}
				mocks.issuer.EXPECT().
					Issue(gomock.Any(), expectedClaims, gomock.Any()).
					Times(1).Return(&tokens.TokenInfo{Token: "the quick brown fox jumps over the lazy dog"}, nil)
			},
			expectedRsp: &v1.GenerateTokenForPermissionsAndScopeResponse{
				Token: "the quick brown fox jumps over the lazy dog",
			},
		},
		"lifetime capping": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   deploymentPermission,
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
				// 300 seconds requested, but policy caps at 10 seconds.
				Lifetime: testExpirationDuration,
			},
			policy: newTokenPolicy(10*time.Second, map[string]v1.Access{
				"Deployment": v1.Access_READ_ACCESS,
			}),
			setup: func(st *testing.T, mocks *mockContainer) {
				setClusterStoreExpectations(mocks.clusterStore, clusterIDNameMap)
				cappedExpiry := testClock().Add(10 * time.Second)
				expectedClaims := tokens.RoxClaims{
					InternalRoles: []*tokens.InternalRole{
						{
							RoleName:      internalRoleName,
							ReadResources: []string{deploymentResource},
							ClusterScopes: []*tokens.ClusterScope{
								{
									ClusterName: testSensorClusterID,
									Namespaces:  []string{"namespace A"},
								},
							},
						},
					},
					Name: fmt.Sprintf(
						claimNameFormat,
						internalRoleName,
						cappedExpiry.Format(time.RFC3339Nano),
					),
				}
				mocks.issuer.EXPECT().
					Issue(gomock.Any(), expectedClaims, gomock.Any()). // TODO: use some form of validation of the withExpiry parameters
					Times(1).Return(&tokens.TokenInfo{Token: "capped-token"}, nil)
			},
			expectedRsp: &v1.GenerateTokenForPermissionsAndScopeResponse{
				Token: "capped-token",
			},
		},
		"success - with custom requester": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions:   deploymentPermission,
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
				Lifetime:      testExpirationDuration,
				// The requester information should be propagated to the claims
				// passed to the issuers.
				Requester: "custom requester",
			},
			policy: permissivePolicy,
			setup: func(t *testing.T, mocks *mockContainer) {
				setClusterStoreExpectations(mocks.clusterStore, clusterIDNameMap)
				expectedClaims := tokens.RoxClaims{
					InternalRoles: []*tokens.InternalRole{
						{
							RoleName:      internalRoleName,
							ReadResources: []string{deploymentResource},
							ClusterScopes: []*tokens.ClusterScope{
								{
									ClusterName: testSensorClusterID,
									Namespaces:  []string{"namespace A"},
								},
							},
						},
					},
					Name: fmt.Sprintf(
						"Generated claims for role %s expiring at %s",
						internalRoleName,
						testTokenExpiry.Format(time.RFC3339Nano),
					),
					Requester: "custom requester",
				}
				mocks.issuer.EXPECT().
					Issue(gomock.Any(), expectedClaims, gomock.Any()).
					Times(1).Return(&tokens.TokenInfo{Token: "the quick brown fox jumps over the lazy dog"}, nil)
			},
			expectedRsp: &v1.GenerateTokenForPermissionsAndScopeResponse{
				Token: "the quick brown fox jumps over the lazy dog",
			},
		},
		"success - no requested permissions": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				// No requested permission results in empty permissions for the user role in the token claims.
				Permissions:   nil,
				ClusterScopes: []*v1.ClusterScope{requestSingleNamespace},
				Lifetime:      testExpirationDuration,
			},
			policy: permissivePolicy,
			setup: func(t *testing.T, mocks *mockContainer) {
				setClusterStoreExpectations(mocks.clusterStore, clusterIDNameMap)
				expectedClaims := tokens.RoxClaims{
					InternalRoles: []*tokens.InternalRole{
						{
							RoleName: internalRoleName,
							ClusterScopes: []*tokens.ClusterScope{
								{
									ClusterName: testSensorClusterID,
									Namespaces:  []string{"namespace A"},
								},
							},
						},
					},
					Name: fmt.Sprintf(
						"Generated claims for role %s expiring at %s",
						internalRoleName,
						testTokenExpiry.Format(time.RFC3339Nano),
					),
				}
				mocks.issuer.EXPECT().
					Issue(gomock.Any(), expectedClaims, gomock.Any()).
					Times(1).Return(&tokens.TokenInfo{Token: "In the days when everybody started fair, the Leopard lived in a place called the High Veldt."}, nil)
			},
			expectedRsp: &v1.GenerateTokenForPermissionsAndScopeResponse{
				Token: "In the days when everybody started fair, the Leopard lived in a place called the High Veldt.",
			},
		},
		"success - no requested scope": {
			input: &v1.GenerateTokenForPermissionsAndScopeRequest{
				Permissions: deploymentPermission,
				// No requested scope results in empty scope for the user role in the token claims.
				ClusterScopes: nil,
				Lifetime:      testExpirationDuration,
			},
			policy: permissivePolicy,
			setup: func(t *testing.T, mocks *mockContainer) {
				expectedClaims := tokens.RoxClaims{
					InternalRoles: []*tokens.InternalRole{
						{
							RoleName:      internalRoleName,
							ReadResources: []string{deploymentResource},
							ClusterScopes: nil,
						},
					},
					Name: fmt.Sprintf(
						"Generated claims for role %s expiring at %s",
						internalRoleName,
						testTokenExpiry.Format(time.RFC3339Nano),
					),
				}
				mocks.issuer.EXPECT().
					Issue(gomock.Any(), expectedClaims, gomock.Any()).
					Times(1).Return(&tokens.TokenInfo{Token: "In the high and far-off times the elephant had no trunk."}, nil)
			},
			expectedRsp: &v1.GenerateTokenForPermissionsAndScopeResponse{
				Token: "In the high and far-off times the elephant had no trunk.",
			},
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
			policy: newTokenPolicy(300*time.Second, map[string]v1.Access{
				"Deployment": v1.Access_READ_ACCESS,
				"Image":      v1.Access_READ_WRITE_ACCESS,
			}),
			setup: func(t *testing.T, mocks *mockContainer) {
				setClusterStoreExpectations(mocks.clusterStore, clusterIDNameMap)
				setClusterStoreExpectations(mocks.clusterStore, clusterIDNameMap)
				expectedClaims := tokens.RoxClaims{
					InternalRoles: []*tokens.InternalRole{
						{
							RoleName:       internalRoleName,
							ReadResources:  []string{deploymentResource},
							WriteResources: []string{imageResource},
							ClusterScopes: []*tokens.ClusterScope{
								{
									ClusterName: testSensorClusterID,
									Namespaces:  []string{"namespace A"},
								},
								{
									ClusterName:       testSensorClusterID,
									ClusterFullAccess: true,
								},
							},
						},
					},
					Name: fmt.Sprintf(
						"Generated claims for role %s expiring at %s",
						internalRoleName,
						testTokenExpiry.Format(time.RFC3339Nano),
					),
				}
				mocks.issuer.EXPECT().
					Issue(gomock.Any(), expectedClaims, gomock.Any()).
					Times(1).
					Return(&tokens.TokenInfo{Token: "Hear and attend and listen; for this befell and happened and became and was, when the Tame animals were wild."}, nil)
			},
			expectedRsp: &v1.GenerateTokenForPermissionsAndScopeResponse{
				Token: "Hear and attend and listen; for this befell and happened and became and was, when the Tame animals were wild.",
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			mocks := &mockContainer{
				clusterStore: clusterDataStoreMocks.NewMockDataStore(mockCtrl),
				roleStore:    roleDataStoreMocks.NewMockDataStore(mockCtrl),
				issuer:       tokensMocks.NewMockIssuer(mockCtrl),
			}
			svc := createService(it, mocks.issuer, mocks.clusterStore, mocks.roleStore, tc.policy)
			if tc.setup != nil {
				tc.setup(it, mocks)
			}
			ctx := sensorContext(it, mockCtrl, fixtureconsts.Cluster1)

			rsp, err := svc.GenerateTokenForPermissionsAndScope(ctx, tc.input)
			assert.NotNil(it, rsp)
			protoassert.Equal(it, tc.expectedRsp, rsp)
			assert.NoError(it, err)
		})
	}
}

func setClusterStoreExpectations(
	mockClusterStore *clusterDataStoreMocks.MockDataStore,
	clusterIDNameMap map[string]string,
) {
	for clusterID, clusterName := range clusterIDNameMap {
		mockClusterStore.EXPECT().
			GetClusterName(gomock.Any(), clusterID).
			Times(1).
			Return(clusterName, true, nil)
	}
}
