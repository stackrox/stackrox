package basic

import (
	"encoding/base64"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/htpasswd"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testUsername = "test-username"
	testPassword = "test-password"

	testRoleName = "test-role"
)

var (
	errDummy = errors.New("dummy test error")

	testRolePermissions = map[string]storage.Access{
		"Deployment": storage.Access_READ_ACCESS,
	}
)

func makeBasicAuthHeader(username, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

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

func newTestExtractor(t *testing.T, mockCtrl *gomock.Controller, roleMapper permissions.RoleMapper, authProvider authproviders.Provider) *Extractor {
	hashFile := newTestHashFile(t, testUsername, testPassword)
	extractor, err := NewExtractor(NewManager(hashFile, roleMapper), authProvider)
	require.NoError(t, err)
	return extractor
}

func newTestRole() *testRole {
	return &testRole{
		name:        testRoleName,
		permissions: testRolePermissions,
		accessScope: accessScopeIncludeAll,
	}
}
