package basic

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	authProviderMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestIdentity(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	authProvider := authProviderMocks.NewMockProvider(mockCtrl)
	testRole := newTestRole()
	id := identity{
		username:      testUsername,
		resolvedRoles: []permissions.ResolvedRole{testRole},
		authProvider:  authProvider,
	}

	var _ authn.Identity = id
	var _ Identity = id

	assert.Equal(t, testUsername, id.UID())
	assert.Equal(t, testUsername, id.FriendlyName())
	assert.Equal(t, testUsername, id.FullName())
	assert.Equal(t, testRolePermissions, id.Permissions())
	assert.Equal(t, []permissions.ResolvedRole{testRole}, id.Roles())
	protoassert.Equal(t, &storage.UserInfo{
		Username:    testUsername,
		Permissions: &storage.UserInfo_ResourceToAccess{ResourceToAccess: testRolePermissions},
		Roles:       []*storage.UserInfo_Role{{Name: testRoleName, ResourceToAccess: testRolePermissions}},
	}, id.User())
	assert.Nil(t, id.Service())
	assert.Equal(t, map[string][]string{
		"username": {testUsername},
		"role":     {testRoleName},
	}, id.Attributes())
	validFrom, validUntil := id.ValidityPeriod()
	assert.Zero(t, validFrom)
	assert.Equal(t, timeutil.MaxProtoValid, validUntil)
	assert.Same(t, authProvider, id.ExternalAuthProvider())
}

func TestIdentityAsExternalUser(t *testing.T) {
	testRole := newTestRole()
	id := identity{
		username:      testUsername,
		resolvedRoles: []permissions.ResolvedRole{testRole},
	}

	assert.Equal(t, &tokens.ExternalUserClaim{
		UserID: testUsername,
		Attributes: map[string][]string{
			"username": {testUsername},
			"role":     {testRoleName},
		},
	}, id.AsExternalUser())
}

func TestIsBasicIdentity(t *testing.T) {
	t.Run("basic identity", func(it *testing.T) {
		assert.True(it, IsBasicIdentity(identity{username: testUsername}))
	})

	t.Run("non-basic identity", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		otherIdentity := mocks.NewMockIdentity(mockCtrl)
		assert.False(it, IsBasicIdentity(otherIdentity))
	})

	t.Run("nil identity", func(it *testing.T) {
		var nilIdentity authn.Identity
		assert.False(it, IsBasicIdentity(nilIdentity))
	})
}
