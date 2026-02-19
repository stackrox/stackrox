package tokenbased

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	authProviderMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	tokenMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	mockSourceID  = "mock source ID"
	mockSourceID2 = "mock source ID 2"

	cluster1 = "cluster 1"
	cluster2 = "cluster 2"

	namespaceA = "namespaceA"

	deploymentResource = "Deployment"
	imageResource      = "Image"

	readAccess      = "READ_ACCESS"
	readWriteAccess = "READ_WRITE_ACCESS"

	internalRoleName = "internal role"

	mockAuthProviderID   = "mock auth provider ID"
	mockAuthProviderName = "mock auth provider name"
	mockAuthProviderType = "mock auth provider type"
)

type testIdentity struct {
	uid          string
	friendlyName string
	fullName     string
	user         *storage.UserInfo
	permissions  map[string]storage.Access
	roles        []permissions.ResolvedRole
	attributes   map[string][]string
	expiry       time.Time
	authProvider authproviders.Provider
}

func TestCreateRoleBasedIdentity(t *testing.T) {
	const externalUserID1 = "external user ID 1"

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockSource := tokenMocks.NewMockSource(mockCtrl)
	mockSource.EXPECT().ID().AnyTimes().Return(mockSourceID)
	mockSource2 := tokenMocks.NewMockSource(mockCtrl)
	mockSource2.EXPECT().ID().AnyTimes().Return(mockSourceID2)

	mockAuthProvider := authProviderMocks.NewMockProvider(mockCtrl)
	mockAuthProvider.EXPECT().ID().AnyTimes().Return(mockAuthProviderID)
	mockAuthProvider.EXPECT().Name().AnyTimes().Return(mockAuthProviderName)
	mockAuthProvider.EXPECT().Type().AnyTimes().Return(mockAuthProviderType)
	mockAuthProvider.EXPECT().StorageView().AnyTimes().Return(nil)

	role1 := &tokens.InternalRole{
		Permissions:   map[string]string{deploymentResource: readAccess},
		ClusterScopes: []*tokens.ClusterScope{{ClusterName: cluster1, ClusterFullAccess: true}},
	}
	role2 := &tokens.InternalRole{
		Permissions:   map[string]string{imageResource: readWriteAccess},
		ClusterScopes: []*tokens.ClusterScope{{ClusterName: cluster2, Namespaces: []string{namespaceA}}},
	}

	role1Permissions := map[string]storage.Access{
		deploymentResource: storage.Access_READ_ACCESS,
	}
	role2Permissions := map[string]storage.Access{
		imageResource: storage.Access_READ_WRITE_ACCESS,
	}
	bothRolePermissions := map[string]storage.Access{
		deploymentResource: storage.Access_READ_ACCESS,
		imageResource:      storage.Access_READ_WRITE_ACCESS,
	}
	for name, tc := range map[string]struct {
		roles            []permissions.ResolvedRole
		token            *tokens.TokenInfo
		authProvider     authproviders.Provider
		expectedIdentity *testIdentity
	}{
		"minimal inputs": {
			roles: nil,
			token: &tokens.TokenInfo{
				Claims: &tokens.Claims{
					RoxClaims: tokens.RoxClaims{
						ExternalUser: &tokens.ExternalUserClaim{},
					},
				},
				Sources: []tokens.Source{mockSource},
			},
			authProvider: nil,
			expectedIdentity: &testIdentity{
				uid: fmt.Sprintf("sso:%s:%s", mockSourceID, ""),
				user: &storage.UserInfo{
					Permissions: &storage.UserInfo_ResourceToAccess{},
					Roles:       make([]*storage.UserInfo_Role, 0),
				},
				expiry: timeutil.Max,
			},
		},
		"UID construction takes the ID of the first source and the token external user ID": {
			roles: nil,
			token: &tokens.TokenInfo{
				Claims: &tokens.Claims{
					RoxClaims: tokens.RoxClaims{
						ExternalUser: &tokens.ExternalUserClaim{
							UserID: externalUserID1,
						},
					},
				},
				Sources: []tokens.Source{mockSource2, mockSource},
			},
			authProvider: nil,
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("sso:%s:%s", mockSourceID2, externalUserID1),
				friendlyName: externalUserID1,
				user: &storage.UserInfo{
					FriendlyName: externalUserID1,
					Permissions:  &storage.UserInfo_ResourceToAccess{},
					Roles:        make([]*storage.UserInfo_Role, 0),
				},
				expiry: timeutil.Max,
			},
		},
		"roles are propagated from input and used for identity permissions": {
			roles: []permissions.ResolvedRole{role1, role2},
			token: &tokens.TokenInfo{
				Claims: &tokens.Claims{
					RoxClaims: tokens.RoxClaims{
						ExternalUser: &tokens.ExternalUserClaim{
							UserID: externalUserID1,
						},
					},
				},
				Sources: []tokens.Source{mockSource},
			},
			authProvider: nil,
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("sso:%s:%s", mockSourceID, externalUserID1),
				friendlyName: externalUserID1,
				permissions:  bothRolePermissions,
				roles:        []permissions.ResolvedRole{role1, role2},
				user: &storage.UserInfo{
					FriendlyName: externalUserID1,
					Permissions: &storage.UserInfo_ResourceToAccess{
						ResourceToAccess: bothRolePermissions,
					},
					Roles: []*storage.UserInfo_Role{
						{
							Name:             internalRoleName,
							ResourceToAccess: role1Permissions,
						},
						{
							Name:             internalRoleName,
							ResourceToAccess: role2Permissions,
						},
					},
				},
				expiry: timeutil.Max,
			},
		},
		"authProvider is propagated from input and used for identity ExternalAuthProvider": {
			roles: []permissions.ResolvedRole{role1},
			token: &tokens.TokenInfo{
				Claims: &tokens.Claims{
					RoxClaims: tokens.RoxClaims{
						ExternalUser: &tokens.ExternalUserClaim{
							UserID: externalUserID1,
						},
					},
				},
				Sources: []tokens.Source{mockSource},
			},
			authProvider: mockAuthProvider,
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("sso:%s:%s", mockSourceID, externalUserID1),
				friendlyName: externalUserID1,
				permissions:  role1Permissions,
				roles:        []permissions.ResolvedRole{role1},
				user: &storage.UserInfo{
					FriendlyName: externalUserID1,
					Permissions: &storage.UserInfo_ResourceToAccess{
						ResourceToAccess: role1Permissions,
					},
					Roles: []*storage.UserInfo_Role{
						{
							Name:             internalRoleName,
							ResourceToAccess: role1Permissions,
						},
					},
				},
				expiry:       timeutil.Max,
				authProvider: mockAuthProvider,
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			identity := createRoleBasedIdentity(tc.roles, tc.token, tc.authProvider)
			validateIdentity(it, tc.expectedIdentity, identity)
		})
	}

	t.Run("nil token leads to panic", func(it *testing.T) {
		assert.Panics(it, func() {
			_ = createRoleBasedIdentity(nil, nil, nil)
		})
	})

	t.Run("token with nil external user leads to panic", func(it *testing.T) {
		assert.Panics(it, func() {
			token := &tokens.TokenInfo{
				Sources: []tokens.Source{mockSource},
			}
			_ = createRoleBasedIdentity(nil, token, nil)
		})
	})

	t.Run("token without at least one source leads to panic", func(it *testing.T) {
		assert.Panics(it, func() {
			token := &tokens.TokenInfo{
				Claims: &tokens.Claims{RoxClaims: tokens.RoxClaims{ExternalUser: &tokens.ExternalUserClaim{}}},
			}
			_ = createRoleBasedIdentity(nil, token, nil)
		})
	})
}

func validateIdentity(t testing.TB, expected *testIdentity, actual authn.Identity) {
	assert.Equal(t, expected.uid, actual.UID())
	assert.Equal(t, expected.friendlyName, actual.FriendlyName())
	assert.Equal(t, expected.fullName, actual.FullName())
	protoassert.Equal(t, expected.user, actual.User())
	assert.Equal(t, expected.permissions, actual.Permissions())
	validateRoles(t, expected.roles, actual.Roles())
	assert.Nil(t, actual.Service())
	assert.Equal(t, expected.attributes, actual.Attributes())
	validFrom, validUntil := actual.ValidityPeriod()
	assert.Zero(t, validFrom)
	assert.Equal(t, expected.expiry, validUntil)
	// validate external auth provider
	assert.Equal(t, expected.authProvider == nil, actual.ExternalAuthProvider() == nil)
	if expected.authProvider != nil && actual.ExternalAuthProvider() != nil {
		assert.Equal(t, expected.authProvider.ID(), actual.ExternalAuthProvider().ID())
		assert.Equal(t, expected.authProvider.Name(), actual.ExternalAuthProvider().Name())
		assert.Equal(t, expected.authProvider.Type(), actual.ExternalAuthProvider().Type())
		protoassert.Equal(t, expected.authProvider.StorageView(), actual.ExternalAuthProvider().StorageView())
	}
}

func validateRoles(t testing.TB, expected []permissions.ResolvedRole, actual []permissions.ResolvedRole) {
	assert.Equal(t, len(expected), len(actual))
	if len(expected) != len(actual) {
		return
	}
	for i := range expected {
		expectedRole := expected[i]
		actualRole := actual[i]
		assert.Equal(t, expectedRole.GetRoleName(), actualRole.GetRoleName())
		assert.Equal(t, expectedRole.GetPermissions(), actualRole.GetPermissions())
		protoassert.Equal(t, expectedRole.GetAccessScope(), actualRole.GetAccessScope())
	}
}
