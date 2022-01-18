package authproviders

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
	perm "github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

/********
* Tests *
********/

func TestRegistryProviderCallback(t *testing.T) {
	suite.Run(t, new(registryProviderCallbackTestSuite))
}

type registryProviderCallbackTestSuite struct {
	suite.Suite

	registry *registryImpl
	err      error
	ctx      context.Context
	writer   *httptest.ResponseRecorder
	assert   *assert.Assertions
}

func (s *registryProviderCallbackTestSuite) SetupTest() {
	s.err = nil
	s.assert = assert.New(s.T())
	testAuthProviderStore := &tstAuthProviderStore{}
	testRoleMapperFactory := &tstRoleMapperFactory{}
	testTokenIssuerFactory := &tstTokenIssuerFactory{}
	s.registry = &registryImpl{
		ServeMux:      http.NewServeMux(),
		urlPathPrefix: "sssotest",
		redirectURL:   "dummyredirect",
		store:         testAuthProviderStore,
		issuerFactory: testTokenIssuerFactory,

		backendFactories: make(map[string]BackendFactory),
		providers:        make(map[string]Provider),

		roleMapperFactory: testRoleMapperFactory,
	}
	s.ctx = context.Background()
	s.err = s.registry.RegisterBackendFactory(s.ctx, dummyProviderType, newTestAuthProviderBackendFactory)
	s.assert.Equal(nil, s.err, "backend registration at SetupTest should not trigger errors")
	s.err = s.registry.Init()
	s.assert.Equal(nil, s.err, "registry initialization at SetupTest should not trigger errors")
	s.writer = httptest.NewRecorder()
}

func (s *registryProviderCallbackTestSuite) TearDownTest() {
	testAuthProviderBackendFactory.registerProcessResponse("", "", nil)
	testAuthProviderBackend.registerProcessHTTPResponse(nil, nil)
	testRoleMapper.registerRoleMapping(nil)
}

func (s *registryProviderCallbackTestSuite) TestBadCallbackURL() {
	req, _ := http.NewRequest(http.MethodGet, "some/bad/URL/path", strings.NewReader(""))
	s.registry.providersHTTPHandler(s.writer, req)
	s.assert.Equal(404, s.writer.Code, "bad path should trigger NotFound error")
}

func (s *registryProviderCallbackTestSuite) TestMissingProviderCallbackURL() {
	urlPrefix := s.registry.providersURLPrefix()
	req, _ := http.NewRequest(http.MethodGet, urlPrefix, strings.NewReader(""))
	s.registry.providersHTTPHandler(s.writer, req)
	s.assert.Equal(404, s.writer.Code, "missing provider callback request should trigger NotFound error")
}

func (s *registryProviderCallbackTestSuite) TestNotRegisteredProviderCallbackURL() {
	urlPrefix := s.registry.providersURLPrefix()
	req, _ := http.NewRequest(http.MethodGet, urlPrefix+"someotherprovider/callback", strings.NewReader(""))
	s.registry.providersHTTPHandler(s.writer, req)
	s.assert.Equal(404, s.writer.Code, "provider callback request to not registered provider type "+
		"should trigger NotFound error")
}

func (s *registryProviderCallbackTestSuite) TestInvalidProviderIDInRequest() {
	providerTypeID := "otherprovider"
	urlPrefix := s.registry.providersURLPrefix()
	req, _ := http.NewRequest(http.MethodGet, urlPrefix+dummyProviderType+"/callback", strings.NewReader(""))
	testAuthProviderBackendFactory.registerProcessResponse(providerTypeID, "", nil)
	s.registry.providersHTTPHandler(s.writer, req)
	s.assert.Equal(303, s.writer.Code, "provider callback request to wrong registered provider type "+
		"should trigger redirect")
	responseHeaders := s.writer.Header()
	redirectURL, _ := url.Parse(responseHeaders.Get("Location"))
	redirectURLFragments, _ := url.ParseQuery(redirectURL.Fragment)
	s.assert.Equal(s.registry.redirectURL, redirectURL.Path, "provider callback request to wrong registered "+
		"provider type should redirect to the registry redirect URL")
	s.assert.Equal(fmt.Sprintf("invalid auth provider ID %q", providerTypeID), redirectURLFragments.Get("error"),
		"provider callback request to wrong registered provider type should issue an explicit error message")
}

func (s *registryProviderCallbackTestSuite) TestAuthProviderBackendParseError() {
	urlPrefix := s.registry.providersURLPrefix()
	req, _ := http.NewRequest(http.MethodGet, urlPrefix+dummyProviderType+"/callback", strings.NewReader(""))
	testAuthProviderBackendFactory.registerProcessResponse(dummyProviderType, "", nil)
	parsingError := errors.New("authprovider backend parsing error message for test")
	testAuthProviderBackend.registerProcessHTTPResponse(nil, parsingError)
	s.registry.providersHTTPHandler(s.writer, req)
	s.assert.Equal(303, s.writer.Code, "error in provider backend request parsing should trigger redirect")
	responseHeaders := s.writer.Header()
	redirectURL, _ := url.Parse(responseHeaders.Get("Location"))
	redirectURLFragments, _ := url.ParseQuery(redirectURL.Fragment)
	s.assert.Equal(s.registry.redirectURL, redirectURL.Path, "error in provider backend request parsing "+
		"should redirect to the registry redirect URL")
	s.assert.Equal(parsingError.Error(), redirectURLFragments.Get("error"),
		"provider callback should propagate the provider backend parsing error if any")
}

func (s *registryProviderCallbackTestSuite) TestAuthProviderBackendParseReturnsEmptyResponse() {
	urlPrefix := s.registry.providersURLPrefix()
	req, _ := http.NewRequest(http.MethodGet, urlPrefix+dummyProviderType+"/callback", strings.NewReader(""))
	testAuthProviderBackendFactory.registerProcessResponse(dummyProviderType, "", nil)
	// Explicitely generate an empty auth response to trigger identity creation error
	var authResponse *AuthResponse
	testAuthProviderBackend.registerProcessHTTPResponse(authResponse, nil)
	identityCreationError := errors.New("authentication response is empty")
	identityCreationError = pkgErrors.Wrap(identityCreationError, "cannot create role based identity")
	s.registry.providersHTTPHandler(s.writer, req)
	s.assert.Equal(303, s.writer.Code, "invalid input for identity creation should trigger redirect")
	responseHeaders := s.writer.Header()
	redirectURL, _ := url.Parse(responseHeaders.Get("Location"))
	redirectURLFragments, _ := url.ParseQuery(redirectURL.Fragment)
	s.assert.Equal(s.registry.redirectURL, redirectURL.Path, "invalid input for identity creation "+
		"should redirect to the registry redirect URL")
	s.assert.Equal(identityCreationError.Error(), redirectURLFragments.Get("error"),
		"provider callback should propagate the identity creation error if any")
}

func (s *registryProviderCallbackTestSuite) TestAuthProviderBackendLoginURLError() {
	loginURL := s.registry.loginURL(dummyProviderType)
	req, _ := http.NewRequest(http.MethodGet, loginURL, strings.NewReader(""))
	testAuthProviderBackend.registerLoginURL("some.url", errors.New("some error"))
	s.registry.loginHTTPHandler(s.writer, req)
	s.assert.Equal(500, s.writer.Code, "login URL should return error")
	body := s.writer.Result().Body
	defer func() {
		_ = body.Close()
	}()
	b, _ := io.ReadAll(body)
	s.assert.Equal("could not get login URL: some error\n", string(b), "login URL should return error")
}

func (s *registryProviderCallbackTestSuite) TestAuthProviderBackendLoginURLEmpty() {
	loginURL := s.registry.loginURL(dummyProviderType)
	req, _ := http.NewRequest(http.MethodGet, loginURL, strings.NewReader(""))
	testAuthProviderBackend.registerLoginURL("", nil)
	s.registry.loginHTTPHandler(s.writer, req)
	s.assert.Equal(500, s.writer.Code, "login URL should return error")
	body := s.writer.Result().Body
	defer func() {
		_ = body.Close()
	}()
	b, _ := io.ReadAll(body)
	s.assert.Equal("empty login URL\n", string(b), "login URL should return error")
}

func (s *registryProviderCallbackTestSuite) TestAuthenticationTestModeUserWithoutRole() {
	urlPrefix := s.registry.providersURLPrefix()
	req, _ := http.NewRequest(http.MethodGet, urlPrefix+dummyProviderType+"/callback", strings.NewReader(""))
	clientState := idputil.AttachTestStateOrEmpty("", true)
	testAuthProviderBackendFactory.registerProcessResponse(dummyProviderType, clientState, nil)
	authRsp := generateAuthResponse(testUserWithoutRole, nil)
	testAuthProviderBackend.registerProcessHTTPResponse(authRsp, nil)
	rolemapping := make(map[string][]perm.ResolvedRole)
	rolemapping[testUserWithoutRole] = []perm.ResolvedRole{}
	testRoleMapper.registerRoleMapping(rolemapping)
	s.registry.providersHTTPHandler(s.writer, req)
	s.assert.Equal(303, s.writer.Code, "callback activated with test mode should trigger redirect")
	responseHeaders := s.writer.Header()
	redirectURL, _ := url.Parse(responseHeaders.Get("Location"))
	redirectURLFragments, _ := url.ParseQuery(redirectURL.Fragment)
	s.assert.Equal(s.registry.redirectURL, redirectURL.Path, "callback activated with test mode "+
		"should redirect to the registry redirect URL")
	s.assert.Equal(strconv.FormatBool(true), redirectURLFragments.Get("test"),
		"callback activated with test mode should have test set to true in the redirect URL metadata")
}

func (s *registryProviderCallbackTestSuite) TestAuthenticationTestModeUserWithRoles() {
	urlPrefix := s.registry.providersURLPrefix()
	req, _ := http.NewRequest(http.MethodGet, urlPrefix+dummyProviderType+"/callback", strings.NewReader(""))
	clientState := idputil.AttachTestStateOrEmpty("", true)
	testAuthProviderBackendFactory.registerProcessResponse(dummyProviderType, clientState, nil)
	authRsp := generateAuthResponse(testUserWithAdminRole, nil)
	testAuthProviderBackend.registerProcessHTTPResponse(authRsp, nil)
	adminRole := roletest.NewResolvedRoleWithGlobalScope(testUserWithAdminRole, nil)
	rolemapping := make(map[string][]perm.ResolvedRole)
	rolemapping[testUserWithAdminRole] = []perm.ResolvedRole{adminRole}
	testRoleMapper.registerRoleMapping(rolemapping)
	s.registry.providersHTTPHandler(s.writer, req)
	s.assert.Equal(303, s.writer.Code, "callback activated with test mode should trigger redirect")
	responseHeaders := s.writer.Header()
	redirectURL, _ := url.Parse(responseHeaders.Get("Location"))
	redirectURLFragments, _ := url.ParseQuery(redirectURL.Fragment)
	s.assert.Equal(s.registry.redirectURL, redirectURL.Path, "callback activated with test mode "+
		"should redirect to the registry redirect URL")
	s.assert.Equal(strconv.FormatBool(true), redirectURLFragments.Get("test"),
		"callback activated with test mode should have test set to true in the redirect URL metadata")
}

func (s *registryProviderCallbackTestSuite) TestAuthenticationRejectsUserWithoutRole() {
	urlPrefix := s.registry.providersURLPrefix()
	req, _ := http.NewRequest(http.MethodGet, urlPrefix+dummyProviderType+"/callback", strings.NewReader(""))
	clientState := idputil.AttachTestStateOrEmpty("", false)
	testAuthProviderBackendFactory.registerProcessResponse(dummyProviderType, clientState, nil)
	authRsp := generateAuthResponse(testUserWithoutRole, nil)
	testAuthProviderBackend.registerProcessHTTPResponse(authRsp, nil)
	rolemapping := make(map[string][]perm.ResolvedRole)
	rolemapping[testUserWithoutRole] = []perm.ResolvedRole{}
	testRoleMapper.registerRoleMapping(rolemapping)
	s.registry.providersHTTPHandler(s.writer, req)
	s.assert.Equal(303, s.writer.Code, "callback activated for user without role should trigger redirect")
	responseHeaders := s.writer.Header()
	redirectURL, _ := url.Parse(responseHeaders.Get("Location"))
	redirectURLFragments, _ := url.ParseQuery(redirectURL.Fragment)
	s.assert.Equal(s.registry.redirectURL, redirectURL.Path, "callback activated for user without role "+
		"should redirect to the registry redirect URL")
	callbackError := fmt.Errorf("Access for this user is not authorized: %w. Please contact a system administrator.",
		errorhelpers.ErrNoValidRole)
	s.assert.Equal(callbackError.Error(), redirectURLFragments.Get("error"),
		"callback activated for user without role should issue an explicit message")
}

func (s *registryProviderCallbackTestSuite) TestAuthenticationIssuesTokenForUserWithRoles() {
	urlPrefix := s.registry.providersURLPrefix()
	req, _ := http.NewRequest(http.MethodGet, urlPrefix+dummyProviderType+"/callback", strings.NewReader(""))
	clientState := idputil.AttachTestStateOrEmpty("", false)
	testAuthProviderBackendFactory.registerProcessResponse(dummyProviderType, clientState, nil)
	authRsp := generateAuthResponse(testUserWithAdminRole, nil)
	testAuthProviderBackend.registerProcessHTTPResponse(authRsp, nil)
	adminRole := roletest.NewResolvedRoleWithGlobalScope(testUserWithAdminRole, nil)
	rolemapping := make(map[string][]perm.ResolvedRole)
	rolemapping[testUserWithAdminRole] = []perm.ResolvedRole{adminRole}
	testRoleMapper.registerRoleMapping(rolemapping)
	s.registry.providersHTTPHandler(s.writer, req)
	s.assert.Equal(303, s.writer.Code, "callback activated for user with valid roles should trigger redirect")
	responseHeaders := s.writer.Header()
	redirectURL, _ := url.Parse(responseHeaders.Get("Location"))
	redirectURLFragments, _ := url.ParseQuery(redirectURL.Fragment)
	s.assert.Equal(s.registry.redirectURL, redirectURL.Path, "callback activated for user with valid roles "+
		"should redirect to the registry redirect URL")
	s.assert.Equal(testDummyTokenData, redirectURLFragments.Get("token"),
		"callback activated for user with valid roles should issue a token")
}

/*****************************************************
* Elements needed for the tests                      *
* - AuthResponse generator                           *
* - Pseudo-mocks for the various required interfaces *
*****************************************************/

const dummyProviderType = "dummy"
const testUserWithoutRole = "testUserWithoutRole"
const testUserWithAdminRole = "testUserWithAdminRole"
const testDummyTokenData = "dummy test token"

var mockAuthProvider = &storage.AuthProvider{
	Id:               dummyProviderType,
	Name:             "dummy auth provider",
	Type:             dummyProviderType,
	UiEndpoint:       "",
	Enabled:          true,
	Config:           nil,
	LoginUrl:         "",
	Validated:        true,
	ExtraUiEndpoints: []string{},
	Active:           true,
}

func generateAuthResponse(user string, userAttr map[string][]string) *AuthResponse {
	return &AuthResponse{
		Claims: &tokens.ExternalUserClaim{
			UserID:     user,
			FullName:   user,
			Email:      user,
			Attributes: userAttr,
		},
		Expiration: time.Now().Add(5 * time.Minute),
	}
}

// Authprovider store (needed for NewStoreBackedRegistry)
type tstAuthProviderStore struct{}

func (*tstAuthProviderStore) GetAllAuthProviders() ([]*storage.AuthProvider, error) {
	return []*storage.AuthProvider{mockAuthProvider}, nil
}

func (*tstAuthProviderStore) AddAuthProvider(_ context.Context, _ *storage.AuthProvider) error {
	return nil
}

func (*tstAuthProviderStore) UpdateAuthProvider(_ context.Context, _ *storage.AuthProvider) error {
	return nil
}

func (*tstAuthProviderStore) RemoveAuthProvider(_ context.Context, _ string) error {
	return nil
}

// Token issuer factory (needed for NewStoreBackedRegistry)

type tstTokenIssuer struct{}

func (*tstTokenIssuer) Issue(_ context.Context, _ tokens.RoxClaims, _ ...tokens.Option) (*tokens.TokenInfo, error) {
	token := &tokens.TokenInfo{
		Token:   testDummyTokenData,
		Claims:  nil,
		Sources: []tokens.Source{},
	}
	return token, nil
}

type tstTokenIssuerFactory struct{}

func (*tstTokenIssuerFactory) CreateIssuer(_ tokens.Source, _ ...tokens.Option) (tokens.Issuer, error) {
	testTokenIssuer := &tstTokenIssuer{}
	return testTokenIssuer, nil
}

func (*tstTokenIssuerFactory) UnregisterSource(_ tokens.Source) error {
	return nil
}

// RoleMapper factory (needed for NewStoreBackedRegistry)
type tstRoleMapper struct {
	roleMapping map[string][]perm.ResolvedRole
}

func (m *tstRoleMapper) registerRoleMapping(mapping map[string][]perm.ResolvedRole) {
	m.roleMapping = mapping
}

func (m *tstRoleMapper) FromUserDescriptor(_ context.Context, u *perm.UserDescriptor) ([]perm.ResolvedRole, error) {
	return m.roleMapping[u.UserID], nil
}

var testRoleMapper = &tstRoleMapper{}

type tstRoleMapperFactory struct{}

func (*tstRoleMapperFactory) GetRoleMapper(_ string) perm.RoleMapper {
	return testRoleMapper
}

// Authentication backend factory (needed by registry.RegisterBackendFactory)
type tstAuthProviderBackend struct {
	authRsp  *AuthResponse
	err      error
	loginURL string
}

func (b *tstAuthProviderBackend) registerProcessHTTPResponse(authRsp *AuthResponse, err error) {
	b.authRsp = authRsp
	b.err = err
}

func (b *tstAuthProviderBackend) registerLoginURL(loginURL string, err error) {
	b.loginURL = loginURL
	b.err = err
}

func (*tstAuthProviderBackend) Config() map[string]string {
	return nil
}

func (b *tstAuthProviderBackend) LoginURL(_ string, r *requestinfo.RequestInfo) (string, error) {
	return b.loginURL, b.err
}

func (*tstAuthProviderBackend) RefreshURL() string {
	return "refresh"
}

func (*tstAuthProviderBackend) OnEnable(provider Provider) {}

func (*tstAuthProviderBackend) OnDisable(provider Provider) {}

func (b *tstAuthProviderBackend) ProcessHTTPRequest(_ http.ResponseWriter,
	_ *http.Request) (*AuthResponse, error) {
	return b.authRsp, b.err
}

func (*tstAuthProviderBackend) ExchangeToken(_ context.Context,
	_ string, _ string) (*AuthResponse, string, error) {
	return nil, "", nil
}

func (*tstAuthProviderBackend) Validate(_ context.Context, _ *tokens.Claims) error {
	return nil
}

var testAuthProviderBackend = &tstAuthProviderBackend{}

type tstAuthProviderBackendFactory struct {
	providerID  string
	clientState string
	err         error
}

func (f *tstAuthProviderBackendFactory) GetSuggestedAttributes() []string {
	panic("not implemented")
}

func (f *tstAuthProviderBackendFactory) registerProcessResponse(providerID string, clientState string, err error) {
	f.providerID = providerID
	f.clientState = clientState
	f.err = err
}

func (*tstAuthProviderBackendFactory) CreateBackend(_ context.Context, _ string, _ []string,
	_ map[string]string) (Backend, error) {
	return testAuthProviderBackend, nil
}

func (f *tstAuthProviderBackendFactory) ProcessHTTPRequest(_ http.ResponseWriter,
	_ *http.Request) (string, string, error) {
	return f.providerID, f.clientState, f.err
}

func (*tstAuthProviderBackendFactory) ResolveProviderAndClientState(_ string) (string, string, error) {
	return "", "", nil
}

func (*tstAuthProviderBackendFactory) RedactConfig(config map[string]string) map[string]string {
	return config
}

func (*tstAuthProviderBackendFactory) MergeConfig(newCfg, oldCfg map[string]string) map[string]string {
	mergedCfg := make(map[string]string, len(newCfg))
	for k, v := range oldCfg {
		mergedCfg[k] = v
	}
	for k, v := range newCfg {
		mergedCfg[k] = v
	}
	return mergedCfg
}

var testAuthProviderBackendFactory = &tstAuthProviderBackendFactory{}

func newTestAuthProviderBackendFactory(_ string) BackendFactory {
	return testAuthProviderBackendFactory
}
