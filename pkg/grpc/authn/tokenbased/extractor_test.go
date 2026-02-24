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

	tokenName    = "test-token-name"
	tokenSubject = "test-token-subject"
	tokenID      = "test-token-id"
	roleName1    = "role1"
	roleName2    = "role2"
)

var (
	errDummy = errors.New("dummy test error")

	testRole1 = &tokens.InternalRole{
		Permissions:   map[string]string{deploymentResource: readAccess},
		ClusterScopes: []*tokens.ClusterScope{{ClusterName: cluster1, ClusterFullAccess: true}},
	}
	testRole2 = &tokens.InternalRole{
		Permissions:   map[string]string{imageResource: readWriteAccess},
		ClusterScopes: []*tokens.ClusterScope{{ClusterName: cluster2, Namespaces: []string{namespaceA}}},
	}

	testRole1Permissions = map[string]storage.Access{
		deploymentResource: storage.Access_READ_ACCESS,
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

func TestExtractorWithRoleNames(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockSource := tokenMocks.NewMockSource(mockCtrl)
	mockSource.EXPECT().ID().AnyTimes().Return(mockSourceID)

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
				Claims:  buildRoleNamesClaims(tokenName, tokenSubject, tokenID, []string{roleName1}, testExpiresAt),
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
				Claims:  buildRoleNamesClaims(tokenName, tokenSubject, tokenID, []string{roleName1, roleName2}, testExpiresAt),
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
		tokenName,
		strings.Join([]string{roleName1}, ","),
		tokenID,
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
				Claims:  buildRoleNamesClaims(tokenName, tokenSubject, tokenID, []string{roleName1}, testExpiresAt),
				Sources: []tokens.Source{mockAuthProvider},
			},
			roleNames: []string{roleName1},
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("auth-token:%s", tokenID),
				fullName:     tokenName,
				friendlyName: tokenSubject,
				permissions:  testRole1Permissions,
				roles:        []permissions.ResolvedRole{testRole1},
				user:         buildUserInfo(emptyUserName, tokenSubject, []permissions.ResolvedRole{testRole1}),
				attributes:   map[string][]string{"role": {internalRoleName}, "name": {tokenName}},
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
				Claims:  buildRoleNamesClaimsWithExternalUser(tokenName, tokenSubject, tokenID, externalUserEmail, []string{roleName1, roleName2}, testExpiresAt),
				Sources: []tokens.Source{mockAuthProvider},
			},
			roleNames: []string{roleName1, roleName2},
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("auth-token:%s", tokenID),
				fullName:     tokenName,
				friendlyName: tokenSubject,
				permissions:  bothTestRolePermissions,
				roles:        []permissions.ResolvedRole{testRole1, testRole2},
				user:         buildUserInfo(externalUserEmail, tokenSubject, []permissions.ResolvedRole{testRole1, testRole2}),
				attributes:   map[string][]string{"role": {internalRoleName, internalRoleName}, "name": {tokenName}},
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
				Claims:  buildRoleNamesClaims(tokenName, "", tokenID, []string{roleName1}, testExpiresAt),
				Sources: []tokens.Source{mockAuthProvider},
			},
			roleNames: []string{roleName1},
			expectedIdentity: &testIdentity{
				uid:          fmt.Sprintf("auth-token:%s", tokenID),
				friendlyName: builtFriendlyName,
				fullName:     tokenName,
				user:         buildUserInfo(emptyUserName, builtFriendlyName, []permissions.ResolvedRole{testRole1}),
				permissions:  testRole1Permissions,
				roles:        []permissions.ResolvedRole{testRole1},
				attributes:   map[string][]string{"role": {internalRoleName}, "name": {tokenName}},
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

	for name, tc := range map[string]struct {
		testToken            *tokens.TokenInfo
		setupMocks           func(*testExtractor)
		expectedErrorMessage string
	}{
		"Error: No token source": {
			testToken:            &tokens.TokenInfo{},
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
