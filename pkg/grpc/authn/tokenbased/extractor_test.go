package tokenbased

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	authProviderMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionMocks "github.com/stackrox/rox/pkg/auth/permissions/mocks"
	"github.com/stackrox/rox/pkg/auth/tokens"
	tokenMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/metadata"
)

const (
	mockSourceID  = "mock source ID"
	mockSourceID2 = "mock source ID 2"

	cluster1 = "cluster 1"
	cluster2 = "cluster 2"

	namespaceA = "namespaceA"

	deploymentResource = "Deployment"
	imageResource      = "Image"

	externalUserID       = "external user ID"
	externalUserFullName = "external user full name"
	externalUserEmail    = "external.user@mail.provider"

	missingFullName = ""
	missingEmail    = ""
	missingUserID   = ""

	emptyUserName = ""

	mockAuthProviderID   = "mock auth provider ID"
	mockAuthProviderName = "mock auth provider name"
	mockAuthProviderType = "mock auth provider type"

	testName    = "test-token-name"
	testSubject = "test-token-subject"
	testID      = "test-token-id"
	roleName1   = "role1"
	roleName2   = "role2"
)

var (
	errDummy = errors.New("dummy test error")

	testRole1Permissions = map[string]storage.Access{
		deploymentResource: storage.Access_READ_ACCESS,
	}
	testRole2Permissions = map[string]storage.Access{
		imageResource: storage.Access_READ_WRITE_ACCESS,
	}
	bothTestRolePermissions = map[string]storage.Access{
		deploymentResource: storage.Access_READ_ACCESS,
		imageResource:      storage.Access_READ_WRITE_ACCESS,
	}

	emptyUser = &storage.UserInfo{
		Permissions: &storage.UserInfo_ResourceToAccess{},
		Roles:       make([]*storage.UserInfo_Role, 0),
	}

	testExpiresAt = time.Date(1981, time.October, 9, 14, 01, 02, 0, time.UTC)
)

func getTestRole1(mockCtrl *gomock.Controller) permissions.ResolvedRole {
	mockTestRole := permissionMocks.NewMockResolvedRole(mockCtrl)
	mockTestRole.EXPECT().GetRoleName().AnyTimes().Return(roleName1)
	mockTestRole.EXPECT().GetPermissions().AnyTimes().Return(testRole1Permissions)
	accessScope := &storage.SimpleAccessScope{
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{cluster1},
		},
	}
	mockTestRole.EXPECT().GetAccessScope().AnyTimes().Return(accessScope)
	return mockTestRole
}

func getTestRole2(mockCtrl *gomock.Controller) permissions.ResolvedRole {
	mockTestRole := permissionMocks.NewMockResolvedRole(mockCtrl)
	mockTestRole.EXPECT().GetRoleName().AnyTimes().Return(roleName2)
	mockTestRole.EXPECT().GetPermissions().AnyTimes().Return(testRole2Permissions)
	accessScope := &storage.SimpleAccessScope{
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				{
					ClusterName:   cluster2,
					NamespaceName: namespaceA,
				},
			},
		},
	}
	mockTestRole.EXPECT().GetAccessScope().AnyTimes().Return(accessScope)
	return mockTestRole
}

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

func TestExtractorIdentityForRequest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockSource := tokenMocks.NewMockSource(mockCtrl)
	mockSource.EXPECT().ID().AnyTimes().Return(mockSourceID)
	mockAuthProvider := createEnabledMockAuthProvider(mockCtrl)

	testRole1 := getTestRole1(mockCtrl)
	testRole2 := getTestRole2(mockCtrl)

	t.Run("Neither identity nor error for token of type different from Bearer ", func(it *testing.T) {
		te := getTestExtractor(it)
		identityExtractor := NewExtractor(te.roleStore, te.tokenValidator)
		ri := requestinfo.RequestInfo{
			Metadata: metadata.MD{"authorization": []string{"ServiceCert dummyTokenData"}},
		}
		id, err := identityExtractor.IdentityForRequest(it.Context(), ri)
		assert.Nil(it, err)
		assert.Nil(it, id)
	})

	makeRequestInfoWithBearerToken := func(token string) requestinfo.RequestInfo {
		return requestinfo.RequestInfo{
			Metadata: metadata.MD{"authorization": []string{"Bearer " + token}},
		}
	}

	for name, tc := range map[string]struct {
		request    requestinfo.RequestInfo
		setupMocks func(*testExtractor)
		errMsg     string
	}{
		"Error: Token validation error is propagated": {
			request: makeRequestInfoWithBearerToken("fail-validation"),
			setupMocks: func(te *testExtractor) {
				te.tokenValidator.EXPECT().
					Validate(gomock.Any(), "fail-validation").
					Times(1).
					Return(nil, errDummy)
			},
			errMsg: "token validation failed",
		},
		"Error: Missing source": {
			request: makeRequestInfoWithBearerToken("missing-source"),
			setupMocks: func(te *testExtractor) {
				te.tokenValidator.EXPECT().
					Validate(gomock.Any(), "missing-source").
					Times(1).
					Return(&tokens.TokenInfo{}, nil)
			},
			errMsg: "tokens must originate from exactly one source",
		},
		"Error: Too many token sources": {
			request: makeRequestInfoWithBearerToken("too-many-sources"),
			setupMocks: func(te *testExtractor) {
				te.tokenValidator.EXPECT().
					Validate(gomock.Any(), "too-many-sources").
					Times(1).
					Return(&tokens.TokenInfo{Sources: []tokens.Source{mockSource, mockSource}}, nil)
			},
			errMsg: "tokens must originate from exactly one source",
		},
		"Error: Token source not of AuthProvider type": {
			request: makeRequestInfoWithBearerToken("random-type-source"),
			setupMocks: func(te *testExtractor) {
				te.tokenValidator.EXPECT().
					Validate(gomock.Any(), "random-type-source").
					Times(1).
					Return(&tokens.TokenInfo{Sources: []tokens.Source{mockSource}}, nil)
			},
			errMsg: "API tokens must originate from an authentication provider source",
		},
		"Error: Disabled token source": {
			request: makeRequestInfoWithBearerToken("disabled-token-source"),
			setupMocks: func(te *testExtractor) {
				source := authProviderMocks.NewMockProvider(te.mockCtrl)
				source.EXPECT().ID().AnyTimes().Return(mockSourceID)
				source.EXPECT().Name().Times(1).Return(mockAuthProviderName)
				source.EXPECT().Enabled().Times(1).Return(false)
				te.tokenValidator.EXPECT().
					Validate(gomock.Any(), "disabled-token-source").
					Times(1).
					Return(&tokens.TokenInfo{Sources: []tokens.Source{source}}, nil)
			},
			errMsg: fmt.Sprintf("auth provider %q is not enabled", mockAuthProviderName),
		},
		"Error: Token with both RoleName and RoleNames claims": {
			request: makeRequestInfoWithBearerToken("both-role-name-and-role-names"),
			setupMocks: func(te *testExtractor) {
				source := createEnabledMockAuthProvider(te.mockCtrl)
				tokenInfo := &tokens.TokenInfo{
					Sources: []tokens.Source{source},
					Claims: &tokens.Claims{
						RoxClaims: tokens.RoxClaims{
							RoleName:  roleName1,
							RoleNames: []string{roleName1},
						},
					},
				}
				te.tokenValidator.EXPECT().
					Validate(gomock.Any(), "both-role-name-and-role-names").
					Times(1).
					Return(tokenInfo, nil)
			},
			errMsg: "malformed token: uses both 'roles' and deprecated 'role' claims",
		},
		"Error: Failed role resolution for role tokens": {
			request: makeRequestInfoWithBearerToken("failed-role-resolution"),
			setupMocks: func(te *testExtractor) {
				source := createEnabledMockAuthProvider(te.mockCtrl)
				tokenInfo := &tokens.TokenInfo{
					Sources: []tokens.Source{source},
					Claims: &tokens.Claims{
						RoxClaims: tokens.RoxClaims{
							RoleNames: []string{roleName1},
						},
					},
				}
				te.tokenValidator.EXPECT().
					Validate(gomock.Any(), "failed-role-resolution").
					Times(1).
					Return(tokenInfo, nil)
				te.roleStore.EXPECT().GetAndResolveRole(gomock.Any(), roleName1).Times(1).Return(nil, errDummy)
			},
			errMsg: "failed to resolve user roles",
		},
		"Error: Missing role mapper for external user": {
			request: makeRequestInfoWithBearerToken("external-user-missing-role-mapper"),
			setupMocks: func(te *testExtractor) {
				source := createEnabledMockAuthProvider(te.mockCtrl)
				source.EXPECT().RoleMapper().Times(1).Return(nil)
				tokenInfo := &tokens.TokenInfo{
					Sources: []tokens.Source{source},
					Claims: &tokens.Claims{
						RoxClaims: tokens.RoxClaims{
							ExternalUser: &tokens.ExternalUserClaim{},
						},
					},
				}
				te.tokenValidator.EXPECT().
					Validate(gomock.Any(), "external-user-missing-role-mapper").
					Times(1).
					Return(tokenInfo, nil)
			},
			errMsg: "failed to resolve external user",
		},
		"Error: Token with insufficient data": {
			request: makeRequestInfoWithBearerToken("token-with-insufficient-data"),
			setupMocks: func(te *testExtractor) {
				source := createEnabledMockAuthProvider(te.mockCtrl)
				tokenInfo := &tokens.TokenInfo{
					Sources: []tokens.Source{source},
					Claims:  &tokens.Claims{},
				}
				te.tokenValidator.EXPECT().
					Validate(gomock.Any(), "token-with-insufficient-data").
					Times(1).
					Return(tokenInfo, nil)
			},
			errMsg: "could not determine token type",
		},
	} {
		t.Run(name, func(it *testing.T) {
			te := getTestExtractor(it)
			defer te.mockCtrl.Finish()
			if tc.setupMocks != nil {
				tc.setupMocks(te)
			}
			identityExtractor := NewExtractor(te.roleStore, te.tokenValidator)
			id, err := identityExtractor.IdentityForRequest(it.Context(), tc.request)
			assert.Error(it, err)
			assert.ErrorContains(it, err, tc.errMsg)
			assert.Nil(it, id)
		})
	}

	friendlyName := fmt.Sprintf("%s (%s)", externalUserFullName, externalUserEmail)

	for name, tc := range map[string]struct {
		request    requestinfo.RequestInfo
		setupMocks func(*testExtractor)
		identity   *testIdentity
	}{
		"Valid token with role names": {
			request: makeRequestInfoWithBearerToken("valid-token-with-role-names"),
			setupMocks: func(te *testExtractor) {
				te.roleStore.EXPECT().
					GetAndResolveRole(gomock.Any(), roleName1).
					Times(1).
					Return(testRole1, nil)
				te.roleStore.EXPECT().
					GetAndResolveRole(gomock.Any(), roleName2).
					Times(1).
					Return(testRole2, nil)
				setupMockAuthProvider(te.authProvider)
				tokenInfo := &tokens.TokenInfo{
					Sources: []tokens.Source{te.authProvider},
					Claims:  buildRoleNamesClaimsWithExternalUser(testName, testSubject, testID, externalUserEmail, []string{roleName1, roleName2}, testExpiresAt),
				}
				te.tokenValidator.EXPECT().
					Validate(gomock.Any(), "valid-token-with-role-names").
					Times(1).
					Return(tokenInfo, nil)
			},
			identity: &testIdentity{
				uid:          fmt.Sprintf("auth-token:%s", testID),
				fullName:     testName,
				friendlyName: testSubject,
				permissions:  bothTestRolePermissions,
				roles:        []permissions.ResolvedRole{testRole1, testRole2},
				user:         buildUserInfo(externalUserEmail, testSubject, []permissions.ResolvedRole{testRole1, testRole2}),
				attributes:   map[string][]string{"role": {roleName1, roleName2}, "name": {testName}},
				expiry:       testExpiresAt,
				authProvider: mockAuthProvider,
			},
		},
		"Valid token with role name - backward compatibility": {
			request: makeRequestInfoWithBearerToken("valid-token-with-role-name-backward-compatibility"),
			setupMocks: func(te *testExtractor) {
				te.roleStore.EXPECT().
					GetAndResolveRole(gomock.Any(), roleName1).
					Times(1).
					Return(testRole1, nil)
				setupMockAuthProvider(te.authProvider)
				tokenInfo := &tokens.TokenInfo{
					Sources: []tokens.Source{te.authProvider},
					Claims: &tokens.Claims{
						Claims: jwt.Claims{
							Subject:  testSubject,
							ID:       testID,
							IssuedAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
							Expiry:   jwt.NewNumericDate(testExpiresAt),
						},
						RoxClaims: tokens.RoxClaims{
							Name:         testName,
							RoleName:     roleName1,
							ExternalUser: &tokens.ExternalUserClaim{Email: externalUserEmail},
						},
					},
				}
				te.tokenValidator.EXPECT().
					Validate(gomock.Any(), "valid-token-with-role-name-backward-compatibility").
					Times(1).
					Return(tokenInfo, nil)
			},
			identity: &testIdentity{
				uid:          fmt.Sprintf("auth-token:%s", testID),
				fullName:     testName,
				friendlyName: testSubject,
				permissions:  testRole1Permissions,
				roles:        []permissions.ResolvedRole{testRole1},
				user:         buildUserInfo(externalUserEmail, testSubject, []permissions.ResolvedRole{testRole1}),
				attributes:   map[string][]string{"role": {roleName1}, "name": {testName}},
				expiry:       testExpiresAt,
				authProvider: mockAuthProvider,
			},
		},
		"Valid token with external user": {
			request: makeRequestInfoWithBearerToken("valid-token-with-external-user"),
			setupMocks: func(te *testExtractor) {
				roleMapper := permissionMocks.NewMockRoleMapper(te.mockCtrl)
				roleMapper.EXPECT().
					FromUserDescriptor(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]permissions.ResolvedRole{testRole1, testRole2}, nil)
				te.authProvider.EXPECT().RoleMapper().Times(1).Return(roleMapper)
				te.authProvider.EXPECT().MarkAsActive().Times(1).Return(nil)
				setupMockAuthProvider(te.authProvider)
				token := &tokens.TokenInfo{
					Claims: buildExternalUserClaimsWithExpiry(
						externalUserEmail,
						externalUserFullName,
						externalUserID,
						testExpiresAt,
					),
					Sources: []tokens.Source{te.authProvider},
				}
				te.tokenValidator.EXPECT().
					Validate(gomock.Any(), "valid-token-with-external-user").
					Times(1).
					Return(token, nil)
			},
			identity: &testIdentity{
				uid:          fmt.Sprintf("sso:%s:%s", mockAuthProviderID, externalUserID),
				fullName:     externalUserFullName,
				friendlyName: friendlyName,
				permissions:  bothTestRolePermissions,
				roles:        []permissions.ResolvedRole{testRole1, testRole2},
				user:         buildUserInfo(externalUserEmail, friendlyName, []permissions.ResolvedRole{testRole1, testRole2}),
				expiry:       testExpiresAt,
				authProvider: mockAuthProvider,
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			te := getTestExtractor(it)
			defer te.mockCtrl.Finish()
			if tc.setupMocks != nil {
				tc.setupMocks(te)
			}
			identityExtractor := NewExtractor(te.roleStore, te.tokenValidator)
			id, err := identityExtractor.IdentityForRequest(it.Context(), tc.request)
			assert.Nil(it, err)
			validateIdentity(it, tc.identity, id)
		})
	}
}

func TestExtractorWithRoleNames(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockSource := tokenMocks.NewMockSource(mockCtrl)
	mockSource.EXPECT().ID().AnyTimes().Return(mockSourceID)

	testRole1 := getTestRole1(mockCtrl)
	testRole2 := getTestRole2(mockCtrl)

	mockAuthProvider := authProviderMocks.NewMockProvider(mockCtrl)
	setupMockAuthProvider(mockAuthProvider)

	t.Run("nil token leads to panic", func(it *testing.T) {
		te := getTestExtractor(it)
		defer te.mockCtrl.Finish()
		assert.Panics(it, func() {
			_, _ = te.extractor.withRoleNames(it.Context(), nil, nil, mockAuthProvider)
		})
	})

	for name, tc := range map[string]struct {
		testToken            *tokens.TokenInfo
		roleNames            []string
		setupMocks           func(*testExtractor)
		expectedErrorMessage string
	}{
		"Error: Role store GetAndResolveRole fails": {
			testToken: &tokens.TokenInfo{
				Claims:  buildRoleNamesClaims(testName, testSubject, testID, []string{roleName1}, testExpiresAt),
				Sources: []tokens.Source{mockSource},
			},
			roleNames: []string{roleName1},
			setupMocks: func(te *testExtractor) {
				te.roleStore.EXPECT().
					GetAndResolveRole(gomock.Any(), roleName1).
					Times(1).
					Return(nil, errDummy)
			},
			expectedErrorMessage: "failed to read roles",
		},
		"Error: Role store returns error for first of multiple roles": {
			testToken: &tokens.TokenInfo{
				Claims:  buildRoleNamesClaims(testName, testSubject, testID, []string{roleName1, roleName2}, testExpiresAt),
				Sources: []tokens.Source{mockSource},
			},
			roleNames: []string{roleName1, roleName2},
			setupMocks: func(te *testExtractor) {
				te.roleStore.EXPECT().
					GetAndResolveRole(gomock.Any(), roleName1).
					Times(1).
					Return(testRole1, nil)
				te.roleStore.EXPECT().
					GetAndResolveRole(gomock.Any(), roleName2).
					Times(1).
					Return(nil, errDummy)
			},
			expectedErrorMessage: "failed to read roles",
		},
	} {
		t.Run(name, func(it *testing.T) {
			te := getTestExtractor(it)
			defer te.mockCtrl.Finish()
			if tc.setupMocks != nil {
				tc.setupMocks(te)
			}
			identity, extractionError := te.extractor.withRoleNames(it.Context(), tc.testToken, tc.roleNames, mockAuthProvider)
			assert.Nil(it, identity)
			assert.ErrorContains(it, extractionError, tc.expectedErrorMessage)
		})
	}

	builtFriendlyName := fmt.Sprintf(
		"anonymous bearer token %q with roles [%s] (jti: %s, expires: %s)",
		testName,
		strings.Join([]string{roleName1}, ","),
		testID,
		jwt.NewNumericDate(testExpiresAt).Time().Format(time.RFC3339),
	)
	for name, tc := range map[string]struct {
		setupMocks       func(*testExtractor)
		token            *tokens.TokenInfo
		roleNames        []string
		expectedIdentity *testIdentity
	}{
		"Create identity from role names with subject": {
			setupMocks: func(te *testExtractor) {
				te.roleStore.EXPECT().
					GetAndResolveRole(gomock.Any(), roleName1).
					Times(1).
					Return(testRole1, nil)
				setupMockAuthProvider(te.authProvider)
			},
			token: &tokens.TokenInfo{
				Claims:  buildRoleNamesClaims(testName, testSubject, testID, []string{roleName1}, testExpiresAt),
				Sources: []tokens.Source{mockAuthProvider},
			},
			roleNames: []string{roleName1},
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("auth-token:%s", testID),
				fullName:     testName,
				friendlyName: testSubject,
				permissions:  testRole1Permissions,
				roles:        []permissions.ResolvedRole{testRole1},
				user:         buildUserInfo(emptyUserName, testSubject, []permissions.ResolvedRole{testRole1}),
				attributes:   map[string][]string{"role": {roleName1}, "name": {testName}},
				expiry:       testExpiresAt,
				authProvider: mockAuthProvider,
			},
		},
		"Create identity from multiple role names, propagating external user email in identity": {
			setupMocks: func(te *testExtractor) {
				te.roleStore.EXPECT().
					GetAndResolveRole(gomock.Any(), roleName1).
					Times(1).
					Return(testRole1, nil)
				te.roleStore.EXPECT().
					GetAndResolveRole(gomock.Any(), roleName2).
					Times(1).
					Return(testRole2, nil)
				setupMockAuthProvider(te.authProvider)
			},
			token: &tokens.TokenInfo{
				Claims:  buildRoleNamesClaimsWithExternalUser(testName, testSubject, testID, externalUserEmail, []string{roleName1, roleName2}, testExpiresAt),
				Sources: []tokens.Source{mockAuthProvider},
			},
			roleNames: []string{roleName1, roleName2},
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("auth-token:%s", testID),
				fullName:     testName,
				friendlyName: testSubject,
				permissions:  bothTestRolePermissions,
				roles:        []permissions.ResolvedRole{testRole1, testRole2},
				user:         buildUserInfo(externalUserEmail, testSubject, []permissions.ResolvedRole{testRole1, testRole2}),
				attributes:   map[string][]string{"role": {roleName1, roleName2}, "name": {testName}},
				expiry:       testExpiresAt,
				authProvider: mockAuthProvider,
			},
		},
		"Create identity when token has empty subject uses formatted friendly name": {
			setupMocks: func(te *testExtractor) {
				te.roleStore.EXPECT().
					GetAndResolveRole(gomock.Any(), roleName1).
					Times(1).
					Return(testRole1, nil)
				setupMockAuthProvider(te.authProvider)
			},
			token: &tokens.TokenInfo{
				Claims:  buildRoleNamesClaims(testName, "", testID, []string{roleName1}, testExpiresAt),
				Sources: []tokens.Source{mockAuthProvider},
			},
			roleNames: []string{roleName1},
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("auth-token:%s", testID),
				friendlyName: builtFriendlyName,
				fullName:     testName,
				user:         buildUserInfo(emptyUserName, builtFriendlyName, []permissions.ResolvedRole{testRole1}),
				permissions:  testRole1Permissions,
				roles:        []permissions.ResolvedRole{testRole1},
				attributes:   map[string][]string{"role": {roleName1}, "name": {testName}},
				expiry:       testExpiresAt,
				authProvider: mockAuthProvider,
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			te := getTestExtractor(it)
			defer te.mockCtrl.Finish()
			if tc.setupMocks != nil {
				tc.setupMocks(te)
			}
			identity, extractionError := te.extractor.withRoleNames(it.Context(), tc.token, tc.roleNames, mockAuthProvider)
			assert.NoError(it, extractionError)
			validateIdentity(it, tc.expectedIdentity, identity)
		})
	}
}

func TestExtractorWithExternalUser(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockSource := tokenMocks.NewMockSource(mockCtrl)
	mockSource.EXPECT().ID().AnyTimes().Return(mockSourceID)

	testRole1 := getTestRole1(mockCtrl)
	testRole2 := getTestRole2(mockCtrl)

	for name, tc := range map[string]struct {
		testToken            *tokens.TokenInfo
		setupMocks           func(*testExtractor)
		expectedErrorMessage string
	}{
		"Error: No token source": {
			testToken:            &tokens.TokenInfo{},
			expectedErrorMessage: "external user tokens must originate from exactly one source",
		},
		"Error: Too many token sources": {
			testToken:            &tokens.TokenInfo{Sources: []tokens.Source{mockSource, mockSource}},
			expectedErrorMessage: "external user tokens must originate from exactly one source",
		},
		"Error: No role mapper": {
			testToken: &tokens.TokenInfo{Sources: []tokens.Source{mockSource}},
			setupMocks: func(te *testExtractor) {
				te.authProvider.EXPECT().RoleMapper().Times(1).Return(nil)
			},
			expectedErrorMessage: "misconfigured authentication provider: no role mapper defined",
		},
		"Error: Role mapper error": {
			testToken: &tokens.TokenInfo{
				Claims:  buildExternalUserClaims(externalUserEmail, externalUserFullName, externalUserID),
				Sources: []tokens.Source{mockSource},
			},
			setupMocks: func(te *testExtractor) {
				roleMapper := permissionMocks.NewMockRoleMapper(te.mockCtrl)
				roleMapper.EXPECT().
					FromUserDescriptor(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, errDummy)
				te.authProvider.EXPECT().RoleMapper().Times(1).Return(roleMapper)
			},
			expectedErrorMessage: "unable to load role for user",
		},
		"Error: AuthProvider cannot be marked active": {
			testToken: &tokens.TokenInfo{
				Claims:  buildExternalUserClaims(externalUserEmail, externalUserFullName, externalUserID),
				Sources: []tokens.Source{mockSource},
			},
			setupMocks: func(te *testExtractor) {
				roleMapper := permissionMocks.NewMockRoleMapper(te.mockCtrl)
				roleMapper.EXPECT().
					FromUserDescriptor(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, nil)
				te.authProvider.EXPECT().RoleMapper().Times(1).Return(roleMapper)
				te.authProvider.EXPECT().MarkAsActive().Times(1).Return(errDummy)
				te.authProvider.EXPECT().Name().Times(1).Return(mockAuthProviderName)
			},
			expectedErrorMessage: fmt.Sprintf("unable to mark provider %q as validated", mockAuthProviderName),
		},
	} {
		t.Run(name, func(it *testing.T) {
			te := getTestExtractor(it)
			defer te.mockCtrl.Finish()
			if tc.setupMocks != nil {
				tc.setupMocks(te)
			}
			identity, extractionError := te.extractor.withExternalUser(it.Context(), tc.testToken, te.authProvider)
			assert.Nil(it, identity)
			assert.ErrorContains(it, extractionError, tc.expectedErrorMessage)
		})
	}
	t.Run("Create identity from external user", func(it *testing.T) {
		te := getTestExtractor(it)
		defer te.mockCtrl.Finish()
		// setup mocks
		roleMapper := permissionMocks.NewMockRoleMapper(te.mockCtrl)
		roleMapper.EXPECT().
			FromUserDescriptor(gomock.Any(), gomock.Any()).
			Times(1).
			Return([]permissions.ResolvedRole{testRole1, testRole2}, nil)
		te.authProvider.EXPECT().RoleMapper().Times(1).Return(roleMapper)
		te.authProvider.EXPECT().MarkAsActive().Times(1).Return(nil)
		setupMockAuthProvider(te.authProvider)
		// end setup mocks
		token := &tokens.TokenInfo{
			Claims: buildExternalUserClaimsWithExpiry(
				externalUserEmail,
				externalUserFullName,
				externalUserID,
				testExpiresAt,
			),
			Sources: []tokens.Source{mockSource},
		}
		friendlyName := fmt.Sprintf("%s (%s)", externalUserFullName, externalUserEmail)
		expectedIdentity := &testIdentity{
			uid:          fmt.Sprintf("sso:%s:%s", mockSourceID, externalUserID),
			fullName:     externalUserFullName,
			friendlyName: friendlyName,
			permissions:  bothTestRolePermissions,
			roles:        []permissions.ResolvedRole{testRole1, testRole2},
			user:         buildUserInfo(externalUserEmail, friendlyName, []permissions.ResolvedRole{testRole1, testRole2}),
			expiry:       testExpiresAt,
			authProvider: te.authProvider,
		}
		identity, extractionError := te.extractor.withExternalUser(it.Context(), token, te.authProvider)
		assert.NoError(it, extractionError)
		validateIdentity(it, expectedIdentity, identity)
	})
}

func TestCreateRoleBasedIdentity(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockSource := tokenMocks.NewMockSource(mockCtrl)
	mockSource.EXPECT().ID().AnyTimes().Return(mockSourceID)
	mockSource2 := tokenMocks.NewMockSource(mockCtrl)
	mockSource2.EXPECT().ID().AnyTimes().Return(mockSourceID2)

	testRole1 := getTestRole1(mockCtrl)
	testRole2 := getTestRole2(mockCtrl)

	mockAuthProvider := authProviderMocks.NewMockProvider(mockCtrl)
	setupMockAuthProvider(mockAuthProvider)

	for name, tc := range map[string]struct {
		roles            []permissions.ResolvedRole
		token            *tokens.TokenInfo
		authProvider     authproviders.Provider
		expectedIdentity *testIdentity
	}{
		"minimal inputs": {
			roles: nil,
			token: &tokens.TokenInfo{
				Claims:  buildExternalUserClaims(missingEmail, missingFullName, missingUserID),
				Sources: []tokens.Source{mockSource},
			},
			authProvider: nil,
			expectedIdentity: &testIdentity{
				uid:    fmt.Sprintf("sso:%s:%s", mockSourceID, missingUserID),
				user:   emptyUser,
				expiry: timeutil.Max,
			},
		},
		"UID construction takes the ID of the first source and the token external user ID": {
			roles: nil,
			token: &tokens.TokenInfo{
				Claims:  buildExternalUserClaims(missingEmail, missingFullName, externalUserID),
				Sources: []tokens.Source{mockSource2, mockSource},
			},
			authProvider: nil,
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("sso:%s:%s", mockSourceID2, externalUserID),
				friendlyName: externalUserID,
				user:         buildUserInfo(emptyUserName, externalUserID, nil),
				expiry:       timeutil.Max,
			},
		},
		"Friendly name construction takes the provided external user full name when available": {
			roles: nil,
			token: &tokens.TokenInfo{
				Claims:  buildExternalUserClaims(missingEmail, externalUserFullName, externalUserID),
				Sources: []tokens.Source{mockSource},
			},
			authProvider: nil,
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("sso:%s:%s", mockSourceID, externalUserID),
				friendlyName: externalUserFullName,
				fullName:     externalUserFullName,
				user:         buildUserInfo(emptyUserName, externalUserFullName, nil),
				expiry:       timeutil.Max,
			},
		},
		"Friendly name construction adds external user e-mail information when available": {
			roles: nil,
			token: &tokens.TokenInfo{
				Claims:  buildExternalUserClaims(externalUserEmail, externalUserFullName, externalUserID),
				Sources: []tokens.Source{mockSource},
			},
			authProvider: nil,
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("sso:%s:%s", mockSourceID, externalUserID),
				friendlyName: fmt.Sprintf("%s (%s)", externalUserFullName, externalUserEmail),
				fullName:     externalUserFullName,
				user:         buildUserInfo(externalUserEmail, fmt.Sprintf("%s (%s)", externalUserFullName, externalUserEmail), nil),
				expiry:       timeutil.Max,
			},
		},
		"Friendly name construction defaults to external user e-mail information when user full name is not available": {
			roles: nil,
			token: &tokens.TokenInfo{
				Claims:  buildExternalUserClaims(externalUserEmail, missingFullName, externalUserID),
				Sources: []tokens.Source{mockSource},
			},
			authProvider: nil,
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("sso:%s:%s", mockSourceID, externalUserID),
				friendlyName: externalUserEmail,
				user:         buildUserInfo(externalUserEmail, externalUserEmail, nil),
				expiry:       timeutil.Max,
			},
		},
		"Friendly name construction defaults to external user UserID information when neither user full name nor email is not available": {
			roles: nil,
			token: &tokens.TokenInfo{
				Claims:  buildExternalUserClaims(missingEmail, missingFullName, externalUserID),
				Sources: []tokens.Source{mockSource},
			},
			authProvider: nil,
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("sso:%s:%s", mockSourceID, externalUserID),
				friendlyName: externalUserID,
				user:         buildUserInfo(emptyUserName, externalUserID, nil),
				expiry:       timeutil.Max,
			},
		},
		"roles are propagated from input and used for identity permissions": {
			roles: []permissions.ResolvedRole{testRole1, testRole2},
			token: &tokens.TokenInfo{
				Claims:  buildExternalUserClaims(missingEmail, missingFullName, externalUserID),
				Sources: []tokens.Source{mockSource},
			},
			authProvider: nil,
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("sso:%s:%s", mockSourceID, externalUserID),
				friendlyName: externalUserID,
				permissions:  bothTestRolePermissions,
				roles:        []permissions.ResolvedRole{testRole1, testRole2},
				user:         buildUserInfo(emptyUserName, externalUserID, []permissions.ResolvedRole{testRole1, testRole2}),
				expiry:       timeutil.Max,
			},
		},
		"authProvider is propagated from input and used for identity ExternalAuthProvider": {
			roles: []permissions.ResolvedRole{testRole1},
			token: &tokens.TokenInfo{
				Claims:  buildExternalUserClaims(missingEmail, missingFullName, externalUserID),
				Sources: []tokens.Source{mockSource},
			},
			authProvider: mockAuthProvider,
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("sso:%s:%s", mockSourceID, externalUserID),
				friendlyName: externalUserID,
				permissions:  testRole1Permissions,
				roles:        []permissions.ResolvedRole{testRole1},
				user:         buildUserInfo(emptyUserName, externalUserID, []permissions.ResolvedRole{testRole1}),
				expiry:       timeutil.Max,
				authProvider: mockAuthProvider,
			},
		},
		"expiry is propagated from input token claims when available": {
			roles: []permissions.ResolvedRole{testRole1},
			token: &tokens.TokenInfo{
				Claims:  buildExternalUserClaimsWithExpiry(missingEmail, missingFullName, externalUserID, testExpiresAt),
				Sources: []tokens.Source{mockSource},
			},
			authProvider: mockAuthProvider,
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("sso:%s:%s", mockSourceID, externalUserID),
				friendlyName: externalUserID,
				permissions:  testRole1Permissions,
				roles:        []permissions.ResolvedRole{testRole1},
				user:         buildUserInfo(emptyUserName, externalUserID, []permissions.ResolvedRole{testRole1}),
				expiry:       testExpiresAt,
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

	t.Run("token with no source leads to panic", func(it *testing.T) {
		assert.Panics(it, func() {
			token := &tokens.TokenInfo{
				Claims: &tokens.Claims{RoxClaims: tokens.RoxClaims{ExternalUser: &tokens.ExternalUserClaim{}}},
			}
			_ = createRoleBasedIdentity(nil, token, nil)
		})
	})
}

type testExtractor struct {
	mockCtrl       *gomock.Controller
	roleStore      *permissionMocks.MockRoleStore
	tokenValidator *tokenMocks.MockValidator
	authProvider   *authProviderMocks.MockProvider
	extractor      *extractor
}

func getTestExtractor(t *testing.T) *testExtractor {
	mockCtrl := gomock.NewController(t)
	roleStore := permissionMocks.NewMockRoleStore(mockCtrl)
	tokenValidator := tokenMocks.NewMockValidator(mockCtrl)
	return &testExtractor{
		mockCtrl:       mockCtrl,
		roleStore:      roleStore,
		tokenValidator: tokenValidator,
		authProvider:   authProviderMocks.NewMockProvider(mockCtrl),
		extractor: &extractor{
			roleStore: roleStore,
			validator: tokenValidator,
		},
	}
}

func setupMockAuthProvider(provider *authProviderMocks.MockProvider) {
	provider.EXPECT().ID().AnyTimes().Return(mockAuthProviderID)
	provider.EXPECT().Name().AnyTimes().Return(mockAuthProviderName)
	provider.EXPECT().Type().AnyTimes().Return(mockAuthProviderType)
	provider.EXPECT().StorageView().AnyTimes().Return(nil)
	provider.EXPECT().Enabled().AnyTimes().Return(true)
}

func createEnabledMockAuthProvider(mockCtrl *gomock.Controller) *authProviderMocks.MockProvider {
	mockProvider := authProviderMocks.NewMockProvider(mockCtrl)
	setupMockAuthProvider(mockProvider)
	return mockProvider
}

func buildExternalUserClaims(
	userEmail string,
	userFullName string,
	userID string,
) *tokens.Claims {
	return &tokens.Claims{
		RoxClaims: tokens.RoxClaims{
			ExternalUser: &tokens.ExternalUserClaim{
				Email:    userEmail,
				FullName: userFullName,
				UserID:   userID,
			},
		},
	}
}

func buildExternalUserClaimsWithExpiry(
	userEmail string,
	userFullName string,
	userID string,
	expiry time.Time,
) *tokens.Claims {
	return &tokens.Claims{
		Claims: jwt.Claims{
			Expiry: jwt.NewNumericDate(expiry),
		},
		RoxClaims: tokens.RoxClaims{
			ExternalUser: &tokens.ExternalUserClaim{
				Email:    userEmail,
				FullName: userFullName,
				UserID:   userID,
			},
		},
	}
}

func buildRoleNamesClaims(
	name string,
	subject string,
	id string,
	roleNames []string,
	expiry time.Time,
) *tokens.Claims {
	return &tokens.Claims{
		Claims: jwt.Claims{
			Subject:  subject,
			ID:       id,
			IssuedAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			Expiry:   jwt.NewNumericDate(expiry),
		},
		RoxClaims: tokens.RoxClaims{
			Name:      name,
			RoleNames: roleNames,
		},
	}
}

func buildRoleNamesClaimsWithExternalUser(
	name string,
	subject string,
	id string,
	userMail string,
	roleNames []string,
	expiry time.Time,
) *tokens.Claims {
	claimsFromRoles := buildRoleNamesClaims(name, subject, id, roleNames, expiry)
	claimsWithExternalUser := buildExternalUserClaims(userMail, missingFullName, missingUserID)
	claimsFromRoles.RoxClaims.ExternalUser = claimsWithExternalUser.ExternalUser
	return claimsFromRoles
}

func buildUserInfo(userName string, friendlyName string, roles []permissions.ResolvedRole) *storage.UserInfo {
	user := &storage.UserInfo{
		Username:     userName,
		FriendlyName: friendlyName,
		Permissions:  &storage.UserInfo_ResourceToAccess{},
		Roles:        make([]*storage.UserInfo_Role, 0, len(roles)),
	}
	if len(roles) > 0 {
		user.GetPermissions().ResourceToAccess = make(map[string]storage.Access)
	}
	for _, role := range roles {
		for resource, access := range role.GetPermissions() {
			if access > user.GetPermissions().GetResourceToAccess()[resource] {
				user.GetPermissions().GetResourceToAccess()[resource] = access
			}
		}
		user.Roles = append(user.GetRoles(), &storage.UserInfo_Role{
			Name:             role.GetRoleName(),
			ResourceToAccess: role.GetPermissions(),
		})
	}
	return user
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
	assert.Equal(t, expected.expiry.UTC(), validUntil.UTC())
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
