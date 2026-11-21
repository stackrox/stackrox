package basic

import (
	"testing"

	"github.com/stackrox/rox/pkg/auth/htpasswd"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestHashFile(t *testing.T, username, password string) *htpasswd.HashFile {
	hashFile := htpasswd.New()
	require.NoError(t, hashFile.Set(username, password))
	return hashFile
}

func newRoleMapperReturningRole(mockCtrl *gomock.Controller, roles []permissions.ResolvedRole) permissions.RoleMapper {
	roleMapper := permissionMocks.NewMockRoleMapper(mockCtrl)
	roleMapper.EXPECT().
		FromUserDescriptor(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(roles, nil)
	return roleMapper
}

func TestManagerIdentityForCreds(t *testing.T) {
	for name, tc := range map[string]struct {
		username   string
		password   string
		setupMocks func(*gomock.Controller) permissions.RoleMapper
		errMsg     string
		notAuthed  bool
	}{
		"Error: unknown username": {
			username:  "unknown-username",
			password:  testPassword,
			errMsg:    "invalid username and/or password",
			notAuthed: true,
		},
		"Error: wrong password": {
			username:  testUsername,
			password:  "wrong-password",
			errMsg:    "invalid username and/or password",
			notAuthed: true,
		},
		"Error: role mapper fails": {
			username: testUsername,
			password: testPassword,
			setupMocks: func(mockCtrl *gomock.Controller) permissions.RoleMapper {
				roleMapper := permissionMocks.NewMockRoleMapper(mockCtrl)
				roleMapper.EXPECT().
					FromUserDescriptor(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, errDummy)
				return roleMapper
			},
			errMsg: "unable to load roles for user",
		},
	} {
		t.Run(name, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			defer mockCtrl.Finish()
			var roleMapper permissions.RoleMapper
			if tc.setupMocks != nil {
				roleMapper = tc.setupMocks(mockCtrl)
			} else {
				roleMapper = newRoleMapperReturningRole(mockCtrl, nil)
			}
			manager := NewManager(newTestHashFile(it, testUsername, testPassword), roleMapper)

			id, err := manager.IdentityForCreds(it.Context(), tc.username, tc.password, nil)
			assert.Nil(it, id)
			assert.ErrorContains(it, err, tc.errMsg)
			if tc.notAuthed {
				assert.ErrorIs(it, err, errox.NotAuthorized)
			}
		})
	}

	t.Run("Valid credentials yield a basic identity", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		testRole := newTestRole()
		manager := NewManager(
			newTestHashFile(it, testUsername, testPassword),
			newRoleMapperReturningRole(mockCtrl, []permissions.ResolvedRole{testRole}),
		)

		id, err := manager.IdentityForCreds(it.Context(), testUsername, testPassword, nil)
		assert.NoError(it, err)
		require.NotNil(it, id)
		assert.True(it, IsBasicIdentity(id))
		assert.Equal(it, testUsername, id.UID())
		assert.Equal(it, []permissions.ResolvedRole{testRole}, id.Roles())
	})
}

func TestManagerSetHashFile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	manager := NewManager(
		newTestHashFile(t, testUsername, testPassword),
		newRoleMapperReturningRole(mockCtrl, nil),
	)

	_, err := manager.IdentityForCreds(t.Context(), testUsername, testPassword, nil)
	assert.NoError(t, err)

	// After swapping the hash file, only credentials from the new file remain valid.
	manager.SetHashFile(newTestHashFile(t, "other-username", "other-password"))

	_, err = manager.IdentityForCreds(t.Context(), testUsername, testPassword, nil)
	assert.ErrorIs(t, err, errox.NotAuthorized)

	_, err = manager.IdentityForCreds(t.Context(), "other-username", "other-password", nil)
	assert.NoError(t, err)
}
