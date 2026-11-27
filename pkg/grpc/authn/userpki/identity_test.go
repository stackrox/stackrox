package userpki

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	authProviderMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestIdentity(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	authProvider := authProviderMocks.NewMockProvider(mockCtrl)
	testRole := roletest.NewResolvedRole("TestRole", testRolePermissions, nil)
	id := &identity{
		info:          userCert,
		provider:      authProvider,
		resolvedRoles: []permissions.ResolvedRole{testRole},
		attributes:    expectedUserAttributes,
	}

	var _ authn.Identity = id

	assert.Equal(t, "userpki:"+userFingerprint, id.UID())
	assert.Equal(t, "UserA", id.FriendlyName())
	assert.Equal(t, "UserA", id.FullName())
	assert.Equal(t, expectedUserAttributes, id.Attributes())
	assert.Equal(t, []permissions.ResolvedRole{testRole}, id.Roles())
	assert.Equal(t, testRolePermissions, id.Permissions())
	protoassert.Equal(t, &storage.UserInfo{
		FriendlyName: "UserA",
		Permissions:  &storage.UserInfo_ResourceToAccess{ResourceToAccess: testRolePermissions},
		Roles:        []*storage.UserInfo_Role{{Name: "TestRole", ResourceToAccess: testRolePermissions}},
	}, id.User())
	assert.Nil(t, id.Service())
	validFrom, validUntil := id.ValidityPeriod()
	assert.Zero(t, validFrom)
	assert.Equal(t, testNotAfter, validUntil)
	assert.Same(t, authProvider, id.ExternalAuthProvider())
}
