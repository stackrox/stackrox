package basic

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	authProviderMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/auth/htpasswd"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/metadata"
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

func newTestExtractor(t *testing.T, mockCtrl *gomock.Controller, roleMapper permissions.RoleMapper, authProvider authproviders.Provider) *Extractor {
	hashFile := htpasswd.New()
	require.NoError(t, hashFile.Set(testUsername, testPassword))
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

func TestExtractorIdentityForRequest(t *testing.T) {
	// Requests that yield neither identity nor error.
	for name, tc := range map[string]struct {
		requestInfo requestinfo.RequestInfo
	}{
		"NoIdentity-NoError - no authorization metadata": {
			requestInfo: requestinfo.RequestInfo{
				Metadata: metadata.MD{},
			},
		},
		"NoIdentity-NoError - empty authorization metadata": {
			requestInfo: requestinfo.RequestInfo{
				Metadata: metadata.MD{"authorization": []string{""}},
			},
		},
		"NoIdentity-NoError - non-basic authorization metadata": {
			requestInfo: requestinfo.RequestInfo{
				Metadata: metadata.MD{"authorization": []string{"Bearer dummyTokenData"}},
			},
		},
		"NoIdentity-NoError - bare Basic prefix": {
			requestInfo: requestinfo.RequestInfo{
				Metadata: metadata.MD{"authorization": []string{"Basic"}},
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			defer mockCtrl.Finish()
			extractor := newTestExtractor(it, mockCtrl, nil, nil)

			id, err := extractor.IdentityForRequest(it.Context(), tc.requestInfo)
			assert.Nil(it, err)
			assert.Nil(it, id)
		})
	}

	makeRequestInfoWithBasicAuth := func(username, password string) requestinfo.RequestInfo {
		return requestinfo.RequestInfo{
			Metadata: metadata.MD{"authorization": []string{makeBasicAuthHeader(username, password)}},
		}
	}

	// Error paths.
	for name, tc := range map[string]struct {
		request    requestinfo.RequestInfo
		setupMocks func(*gomock.Controller) permissions.RoleMapper
		errMsg     string
	}{
		"Error: malformed basic auth token (invalid base64)": {
			request: requestinfo.RequestInfo{
				Metadata: metadata.MD{"authorization": []string{"Basic ?invalid-base64?"}},
			},
			errMsg: "failed to parse basic auth token",
		},
		"Error: malformed basic auth token (missing colon)": {
			request: requestinfo.RequestInfo{
				Metadata: metadata.MD{"authorization": []string{"Basic " + base64.StdEncoding.EncodeToString([]byte("missing-colon"))}},
			},
			errMsg: "failed to parse basic auth token",
		},
		"Error: unknown username": {
			request: makeRequestInfoWithBasicAuth("unknown-username", testPassword),
			errMsg:  "failed to identify user with username",
		},
		"Error: wrong password": {
			request: makeRequestInfoWithBasicAuth(testUsername, "wrong-password"),
			errMsg:  fmt.Sprintf("failed to identify user with username %q", testUsername),
		},
		"Error: role resolution fails": {
			request: makeRequestInfoWithBasicAuth(testUsername, testPassword),
			setupMocks: func(mockCtrl *gomock.Controller) permissions.RoleMapper {
				roleMapper := permissionMocks.NewMockRoleMapper(mockCtrl)
				roleMapper.EXPECT().
					FromUserDescriptor(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, errDummy)
				return roleMapper
			},
			errMsg: "failed to identify user with username",
		},
	} {
		t.Run(name, func(it *testing.T) {
			mockCtrl := gomock.NewController(it)
			defer mockCtrl.Finish()
			var roleMapper permissions.RoleMapper
			if tc.setupMocks != nil {
				roleMapper = tc.setupMocks(mockCtrl)
			}
			extractor := newTestExtractor(it, mockCtrl, roleMapper, nil)

			id, err := extractor.IdentityForRequest(it.Context(), tc.request)
			assert.ErrorContains(it, err, tc.errMsg)
			assert.Nil(it, id)
		})
	}

	// Invalid credentials should surface as not authorized.
	t.Run("Invalid credentials are not authorized", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()
		extractor := newTestExtractor(it, mockCtrl, nil, nil)

		id, err := extractor.IdentityForRequest(it.Context(), makeRequestInfoWithBasicAuth(testUsername, "wrong-password"))
		assert.ErrorIs(it, err, errox.NotAuthorized)
		assert.Nil(it, id)
	})

	// Success path.
	t.Run("Valid basic auth credentials", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		defer mockCtrl.Finish()

		testRole := newTestRole()
		roleMapper := permissionMocks.NewMockRoleMapper(mockCtrl)
		roleMapper.EXPECT().
			FromUserDescriptor(gomock.Any(), &permissions.UserDescriptor{
				UserID:     testUsername,
				Attributes: map[string][]string{},
			}).
			Times(1).
			Return([]permissions.ResolvedRole{testRole}, nil)
		authProvider := authProviderMocks.NewMockProvider(mockCtrl)
		extractor := newTestExtractor(it, mockCtrl, roleMapper, authProvider)

		id, err := extractor.IdentityForRequest(it.Context(), makeRequestInfoWithBasicAuth(testUsername, testPassword))
		assert.Nil(it, err)
		require.NotNil(it, id)

		assert.True(it, IsBasicIdentity(id))
		assert.Equal(it, testUsername, id.UID())
		assert.Equal(it, testUsername, id.FriendlyName())
		assert.Equal(it, testUsername, id.FullName())
		assert.Equal(it, testRolePermissions, id.Permissions())
		assert.Equal(it, []permissions.ResolvedRole{testRole}, id.Roles())
		protoassert.Equal(it, &storage.UserInfo{
			Username:    testUsername,
			Permissions: &storage.UserInfo_ResourceToAccess{ResourceToAccess: testRolePermissions},
			Roles:       []*storage.UserInfo_Role{{Name: testRoleName, ResourceToAccess: testRolePermissions}},
		}, id.User())
		assert.Nil(it, id.Service())
		assert.Equal(it, map[string][]string{
			"username": {testUsername},
			"role":     {testRoleName},
		}, id.Attributes())
		validFrom, validUntil := id.ValidityPeriod()
		assert.Zero(it, validFrom)
		assert.Equal(it, timeutil.MaxProtoValid, validUntil)
		assert.Same(it, authProvider, id.ExternalAuthProvider())
	})
}
