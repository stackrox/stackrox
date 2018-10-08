package tokenbased

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/tokenbased"
	"github.com/stackrox/rox/pkg/auth/tokenbased/mocks"
	"github.com/stackrox/rox/pkg/auth/tokenbased/user"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/metadata"
)

const (
	fakeMatchingToken = "Bearer FAKETOKEN"
	fakeID            = "FAKEID"
	fakeExpiration    = time.Hour
)

type mockAuthProvider struct {
	V bool
}

func (*mockAuthProvider) Parse(headers map[string][]string, roleMapper tokenbased.RoleMapper) (tokenbased.Identity, error) {
	if headers["authorization"][0] == fakeMatchingToken {
		return tokenbased.NewIdentity(fakeID, nil, time.Now().Add(fakeExpiration)), nil
	}
	return nil, errors.New("invalid token")
}

func (*mockAuthProvider) Enabled() bool {
	return true
}

func (m *mockAuthProvider) Validated() bool {
	return m.V
}

func (*mockAuthProvider) LoginURL() string {
	return ""
}

func (*mockAuthProvider) RefreshURL() string {
	panic("implement me")
}

type mockAuthProviderAccessor struct {
	AuthProviders map[string]authproviders.AuthProvider
	DoNotValidate bool
}

func (m *mockAuthProviderAccessor) GetParsedAuthProviders() map[string]authproviders.AuthProvider {
	return m.AuthProviders
}

func (m *mockAuthProviderAccessor) RecordAuthSuccess(id string) error {
	if m.DoNotValidate {
		return errors.New("i won't validate")
	}
	authProvider, ok := m.AuthProviders[id]
	if !ok {
		panic(fmt.Sprintf("Couldn't find auth provider with id %s", id))
	}
	authProvider.(*mockAuthProvider).V = true
	return nil
}

func newMockAuthProviderAccessor() *mockAuthProviderAccessor {
	return &mockAuthProviderAccessor{
		AuthProviders: make(map[string]authproviders.AuthProvider),
	}
}

type AuthInterceptorTestSuite struct {
	suite.Suite
	authInterceptor          *AuthInterceptor
	mockAuthProviderAccessor *mockAuthProviderAccessor
	mockIdentityParser       *mocks.IdentityParser
}

func (suite *AuthInterceptorTestSuite) SetupTest() {
	suite.mockAuthProviderAccessor = newMockAuthProviderAccessor()
	suite.mockIdentityParser = &mocks.IdentityParser{}
	suite.authInterceptor = NewAuthInterceptor(suite.mockAuthProviderAccessor, nil, suite.mockIdentityParser)
}

func (suite *AuthInterceptorTestSuite) TestAuthInterceptorWithNoProvidersOrTokens() {
	suite.mockIdentityParser.On("Parse", mock.Anything, mock.Anything).Return(nil, errors.New("doesn't exist"))

	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	outGoingCtx, _ := suite.authInterceptor.authToken(ctx)
	identity, err := authn.FromTokenBasedIdentityContext(outGoingCtx)
	suite.Nil(identity)
	suite.Equal(authn.ErrNoContext, err)

	authConfig, err := authn.FromAuthConfigurationContext(outGoingCtx)
	if suite.NoError(err) {
		suite.False(authConfig.ProviderConfigured)
	}
}

func (suite *AuthInterceptorTestSuite) TestAuthInterceptorWithTokensButNoProviders() {
	fakeHeaders := metadata.MD{
		"authorization": {fakeMatchingToken},
	}
	suite.mockIdentityParser.On("Parse", mock.MatchedBy(func(headers map[string][]string) bool {
		return headers["authorization"][0] == fakeMatchingToken
	}), mock.Anything).Return(tokenbased.NewIdentity(fakeID, nil, time.Now().Add(fakeExpiration)), nil)
	ctx := metadata.NewIncomingContext(context.Background(), fakeHeaders)

	outGoingCtx, _ := suite.authInterceptor.authToken(ctx)
	identity, err := authn.FromTokenBasedIdentityContext(outGoingCtx)
	suite.Require().NoError(err)
	suite.Require().NotNil(identity)
	suite.Equal(fakeID, identity.ID())
	suite.True(identity.Expiration().After(time.Now()))
	suite.True(identity.Expiration().Before(time.Now().Add(fakeExpiration + time.Minute)))

	authConfig, err := authn.FromAuthConfigurationContext(outGoingCtx)
	if suite.NoError(err) {
		suite.False(authConfig.ProviderConfigured)
	}
}

func (suite *AuthInterceptorTestSuite) TestAuthInterceptorWithProviders() {
	fakeHeaders := metadata.MD{
		"authorization": {fakeMatchingToken},
	}
	mockAuthProvider := &mockAuthProvider{}

	suite.mockAuthProviderAccessor.AuthProviders = map[string]authproviders.AuthProvider{
		"FAKEID0": mockAuthProvider,
	}

	ctx := metadata.NewIncomingContext(context.Background(), fakeHeaders)

	outGoingCtx, _ := suite.authInterceptor.authToken(ctx)
	identity, err := authn.FromTokenBasedIdentityContext(outGoingCtx)
	suite.Require().NoError(err)
	suite.Require().NotNil(identity)
	suite.Equal(fakeID, identity.ID())
	suite.True(identity.Expiration().After(time.Now()))
	suite.True(identity.Expiration().Before(time.Now().Add(fakeExpiration + time.Minute)))
	suite.Equal(mockAuthProvider, identity.(user.Identity).AuthProvider())

	authConfig, err := authn.FromAuthConfigurationContext(outGoingCtx)
	if suite.NoError(err) {
		suite.True(authConfig.ProviderConfigured)
	}
}

func (suite *AuthInterceptorTestSuite) TestAuthInterceptorWithNonValidatingAuthProvider() {
	fakeHeaders := metadata.MD{
		"authorization": {fakeMatchingToken},
	}
	mockAuthProvider := &mockAuthProvider{}

	suite.mockAuthProviderAccessor.AuthProviders = map[string]authproviders.AuthProvider{
		"FAKEID0": mockAuthProvider,
	}
	suite.mockAuthProviderAccessor.DoNotValidate = true

	ctx := metadata.NewIncomingContext(context.Background(), fakeHeaders)

	outGoingCtx, _ := suite.authInterceptor.authToken(ctx)
	identity, err := authn.FromTokenBasedIdentityContext(outGoingCtx)
	suite.Require().NoError(err)
	suite.Require().NotNil(identity)
	suite.Equal(fakeID, identity.ID())
	suite.True(identity.Expiration().After(time.Now()))
	suite.True(identity.Expiration().Before(time.Now().Add(fakeExpiration + time.Minute)))
	suite.Equal(mockAuthProvider, identity.(user.Identity).AuthProvider())

	authConfig, err := authn.FromAuthConfigurationContext(outGoingCtx)
	if suite.NoError(err) {
		suite.False(authConfig.ProviderConfigured)
	}
}

func (suite *AuthInterceptorTestSuite) TestAuthInterceptorWithErringProvider() {
	suite.mockIdentityParser.On("Parse", mock.Anything, mock.Anything).Return(nil, errors.New("doesn't exist"))

	fakeHeaders := metadata.MD{
		"authorization": {"INVALIDTOKEN"},
	}
	mockAuthProvider := &mockAuthProvider{}

	suite.mockAuthProviderAccessor.AuthProviders = map[string]authproviders.AuthProvider{
		"FAKEID0": mockAuthProvider,
	}

	ctx := metadata.NewIncomingContext(context.Background(), fakeHeaders)

	outGoingCtx, _ := suite.authInterceptor.authToken(ctx)
	identity, err := authn.FromTokenBasedIdentityContext(outGoingCtx)
	suite.Require().Error(err)
	suite.Nil(identity)

	authConfig, err := authn.FromAuthConfigurationContext(outGoingCtx)
	if suite.NoError(err) {
		suite.False(authConfig.ProviderConfigured)
	}
}

func (suite *AuthInterceptorTestSuite) TearDownTest() {
	suite.mockIdentityParser.AssertExpectations(suite.T())
}

func TestAuthInterceptorTestSuite(t *testing.T) {
	suite.Run(t, new(AuthInterceptorTestSuite))
}
