package authproviders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	tokenIssuerMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	authProviderID = "41757468-5072-6f76-6964-657211111111"

	baseLoginURL = "/base/login/url"
	testLoginURL = "/test/login/url"

	baseIssuerID = "546f6b64-6e49-7373-7565-721111111111"
	testIssuerID = "546f6b64-6e49-7373-7565-722222222222"

	baseRoleMapperID = "526f6c65-4d61-7171-6572-111111111111"
	testRoleMapperID = "526f6c65-4d61-7171-6572-222222222222"

	baseAuthProviderBackendID = "4261636b-656e-6478-7878-111111111111"
	testAuthProviderBackendID = "4261636b-656e-6478-7878-222222222222"

	baseAuthProviderBackendFactoryID = "4261636b-656e-6446-6163-746f72791111"
	testAuthProviderBackendFactoryID = "4261636b-656e-6446-6163-746f72792222"
)

var (
	baseAuthProviderBackendConfig = map[string]string{
		"baseKey": "baseValue",
	}

	testAuthProviderBackendConfig = map[string]string{
		"testKey": "testValue",
	}
)

func TestDefaultNewID(t *testing.T) {
	option := DefaultNewID()

	// DefaultNewID should fail on a provider object with nil storedInfo field.
	noStoredInfoProvider := &providerImpl{}
	assert.Nil(t, noStoredInfoProvider.storedInfo)
	_, err := option(noStoredInfoProvider)
	assert.Error(t, err)
	assert.Nil(t, noStoredInfoProvider.storedInfo)

	// DefaultNewID should not overwrite a pre-existing ID,
	// nor should the associated revert action.
	previousIDProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Id: authProviderID,
		},
	}
	assert.Equal(t, authProviderID, previousIDProvider.storedInfo.GetId())
	// apply option
	altRevert, err := option(previousIDProvider)
	assert.NoError(t, err)
	assert.Equal(t, authProviderID, previousIDProvider.storedInfo.GetId())
	_, err = uuid.FromString(previousIDProvider.storedInfo.GetId())
	assert.NoError(t, err)
	// revert option
	err = altRevert(previousIDProvider)
	assert.NoError(t, err)
	assert.Equal(t, authProviderID, previousIDProvider.storedInfo.GetId())

	// DefaultNewID should set the ID field in the storedInfo field with
	// a valid UUID. The returned RevertOption should set the ID
	// back to its previous value (here empty string).
	provider := &providerImpl{}
	provider.storedInfo = &storage.AuthProvider{}
	assert.Empty(t, provider.storedInfo.GetId())
	// apply option
	revert, err := option(provider)
	assert.NoError(t, err)
	assert.NotEmpty(t, provider.storedInfo.GetId())
	_, err = uuid.FromString(provider.storedInfo.GetId())
	assert.NoError(t, err)
	// revert option
	err = revert(provider)
	assert.NoError(t, err)
	assert.Empty(t, provider.storedInfo.GetId())
}

func TestDefaultLoginURL(t *testing.T) {
	called := 0
	getLoginURL := func(_ string) string {
		called += 1
		return testLoginURL
	}
	option := DefaultLoginURL(getLoginURL)

	// DefaultLoginURL should fail on a provider object with nil storedInfo field.
	noStoredInfoProvider := &providerImpl{}
	assert.Nil(t, noStoredInfoProvider.storedInfo)
	_, err := option(noStoredInfoProvider)
	assert.Error(t, err)
	assert.Nil(t, noStoredInfoProvider.storedInfo)
	assert.Zero(t, called)

	// DefaultLoginURL should not overwrite a pre-existing login URL,
	// nor should the associated revert action.
	prevLoginURLProvider := &providerImpl{}
	prevLoginURLProvider.storedInfo = &storage.AuthProvider{
		LoginUrl: baseLoginURL,
	}
	assert.Equal(t, baseLoginURL, prevLoginURLProvider.storedInfo.GetLoginUrl())
	// apply option
	altRevert, err := option(prevLoginURLProvider)
	assert.NoError(t, err)
	assert.Equal(t, baseLoginURL, prevLoginURLProvider.storedInfo.GetLoginUrl())
	assert.Zero(t, called)
	// revert option
	err = altRevert(prevLoginURLProvider)
	assert.NoError(t, err)
	assert.Equal(t, baseLoginURL, prevLoginURLProvider.storedInfo.GetLoginUrl())
	assert.Zero(t, called)

	// DefaultLoginURL should set the login URL to the result
	// of the provided function call, the revert action should reset
	// the login URL to an empty value.
	provider := &providerImpl{}
	provider.storedInfo = &storage.AuthProvider{}
	assert.Empty(t, provider.storedInfo.GetLoginUrl())
	// apply option
	revert, err := option(provider)
	assert.NoError(t, err)
	assert.Equal(t, testLoginURL, provider.storedInfo.GetLoginUrl())
	assert.Equal(t, 1, called)
	// revert option
	err = revert(provider)
	assert.NoError(t, err)
	assert.Empty(t, provider.storedInfo.GetLoginUrl())
	assert.Equal(t, 1, called)
}

func TestDefaultTokenIssuerFromFactory(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	baseIssuer := &fakeIssuer{ID: baseIssuerID}
	testIssuer := &fakeIssuer{ID: testIssuerID}

	mockIssuerFactory := tokenIssuerMocks.NewMockIssuerFactory(mockCtrl)
	mockIssuerFactory.EXPECT().
		CreateIssuer(gomock.Any(), gomock.Any()).
		Times(1).
		Return(testIssuer, nil)

	option := DefaultTokenIssuerFromFactory(mockIssuerFactory)

	// DefaultTokenIssuerFromFactory should not overwrite a pre-existing issuer,
	// nor should the associated revert action.
	previousIssuerProvider := &providerImpl{
		issuer: baseIssuer,
	}
	assert.Equal(t, baseIssuer, previousIssuerProvider.issuer)
	revert2, err := option(previousIssuerProvider)
	assert.Nil(t, err)
	assert.Equal(t, baseIssuer, previousIssuerProvider.issuer)
	err = revert2(previousIssuerProvider)
	assert.NoError(t, err)
	assert.Equal(t, baseIssuer, previousIssuerProvider.issuer)

	// DefaultTokenIssuerFromFactory should set the issuer to the result
	// of the provided factory creation call, the revert action should
	// reset the issuer to a nil value.
	noIssuerProvider := &providerImpl{}
	assert.Nil(t, noIssuerProvider.issuer)
	revert, err := option(noIssuerProvider)
	assert.NoError(t, err)
	assert.Equal(t, testIssuer, noIssuerProvider.issuer)
	err = revert(noIssuerProvider)
	assert.NoError(t, err)
	assert.Nil(t, noIssuerProvider.issuer)
}

func TestDefaultRoleMapperOption(t *testing.T) {
	baseRoleMapper := &fakeRoleMapper{ID: baseRoleMapperID}
	optionTestRoleMapper := &fakeRoleMapper{ID: testRoleMapperID}

	getRoleMapper := func(_ string) permissions.RoleMapper {
		return optionTestRoleMapper
	}
	option := DefaultRoleMapperOption(getRoleMapper)

	// DefaultRoleMapperOption should not erase any pre-existing role mapper,
	// nor should the associated revert action.
	previousMapperProvider := &providerImpl{
		roleMapper: baseRoleMapper,
	}
	assert.Equal(t, baseRoleMapper, previousMapperProvider.roleMapper)
	revert1, err := option(previousMapperProvider)
	assert.NoError(t, err)
	assert.Equal(t, baseRoleMapper, previousMapperProvider.roleMapper)
	err = revert1(previousMapperProvider)
	assert.NoError(t, err)
	assert.Equal(t, baseRoleMapper, previousMapperProvider.roleMapper)

	// DefaultRoleMapperOption should not touch the role mapper if there is
	// no previous mapper but the storage view can NOT provide an ID,
	// nor should the associated revert action.
	noIDProvider := &providerImpl{}
	assert.Nil(t, noIDProvider.roleMapper)
	assert.Empty(t, noIDProvider.storedInfo.GetId())
	revert2, err := option(noIDProvider)
	assert.NoError(t, err)
	assert.Nil(t, noIDProvider.roleMapper)
	err = revert2(noIDProvider)
	assert.NoError(t, err)
	assert.Nil(t, noIDProvider.roleMapper)

	// DefaultRoleMapperOption should set the role mapper if there is
	// no previous mapper and the storage view can provide an ID,
	// the revert action should reset the provider to a nil value.
	provider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Id: authProviderID,
		},
	}
	assert.Nil(t, provider.roleMapper)
	assert.Equal(t, authProviderID, provider.storedInfo.GetId())
	revert3, err := option(provider)
	assert.NoError(t, err)
	assert.Equal(t, optionTestRoleMapper, provider.roleMapper)
	err = revert3(provider)
	assert.NoError(t, err)
	assert.Nil(t, provider.roleMapper)
}

func TestDefaultBackend(t *testing.T) {
	backendFactoryPool := map[string]BackendFactory{
		"test": testAuthProviderBackendFactory,
	}
	option := DefaultBackend(context.Background(), backendFactoryPool)

	baseAuthProviderBackendFactory := &tstAuthProviderBackendFactory{
		ID: baseAuthProviderBackendFactoryID,
	}
	baseAuthProviderBackend := &tstAuthProviderBackend{
		ID: baseAuthProviderBackendID,
	}

	previousBackendProvider := &providerImpl{
		backend:        baseAuthProviderBackend,
		backendFactory: baseAuthProviderBackendFactory,
		storedInfo: &storage.AuthProvider{
			Config: baseAuthProviderBackendConfig,
		},
	}
	assert.Equal(t, baseAuthProviderBackend, previousBackendProvider.backend)
	assert.Equal(t, baseAuthProviderBackendFactory, previousBackendProvider.backendFactory)
	assert.Equal(t, baseAuthProviderBackendConfig, previousBackendProvider.storedInfo.GetConfig())
	revert1, err := option(previousBackendProvider)
	assert.NoError(t, err)
	assert.Equal(t, baseAuthProviderBackend, previousBackendProvider.backend)
	assert.Equal(t, baseAuthProviderBackendFactory, previousBackendProvider.backendFactory)
	assert.Equal(t, baseAuthProviderBackendConfig, previousBackendProvider.storedInfo.GetConfig())
	err = revert1(previousBackendProvider)
	assert.NoError(t, err)
	assert.Equal(t, baseAuthProviderBackend, previousBackendProvider.backend)
	assert.Equal(t, baseAuthProviderBackendFactory, previousBackendProvider.backendFactory)
	assert.Equal(t, baseAuthProviderBackendConfig, previousBackendProvider.storedInfo.GetConfig())

	wrongTypeProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Type: "base",
		},
	}
	assert.Nil(t, wrongTypeProvider.backend)
	assert.Nil(t, wrongTypeProvider.backendFactory)
	assert.Empty(t, wrongTypeProvider.storedInfo.GetConfig())
	revert2, err := option(wrongTypeProvider)
	assert.Error(t, err)
	assert.Nil(t, wrongTypeProvider.backend)
	assert.Nil(t, wrongTypeProvider.backendFactory)
	assert.Empty(t, wrongTypeProvider.storedInfo.GetConfig())
	err = revert2(wrongTypeProvider)
	assert.NoError(t, err)
	assert.Nil(t, wrongTypeProvider.backend)
	assert.Nil(t, wrongTypeProvider.backendFactory)
	assert.Empty(t, wrongTypeProvider.storedInfo.GetConfig())

	provider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Type:   "test",
			Config: baseAuthProviderBackendConfig,
		},
		backendFactory: baseAuthProviderBackendFactory,
	}
	assert.Nil(t, provider.backend)
	assert.Equal(t, baseAuthProviderBackendConfig, provider.storedInfo.GetConfig())
	assert.Equal(t, baseAuthProviderBackendFactory, provider.backendFactory)
	revert3, err := option(provider)
	assert.NoError(t, err)
	assert.Equal(t, testAuthProviderBackend, provider.backend)
	assert.Equal(t, testAuthProviderBackendFactory, provider.backendFactory)
	assert.Equal(t, testAuthProviderBackendConfig, provider.storedInfo.GetConfig())
	err = revert3(provider)
	assert.NoError(t, err)
	assert.Nil(t, provider.backend)
	assert.Equal(t, baseAuthProviderBackendConfig, provider.storedInfo.GetConfig())
	assert.Equal(t, baseAuthProviderBackendFactory, provider.backendFactory)
}

// region test utilities

type fakeIssuer struct {
	ID string
}

func (i *fakeIssuer) Issue(_ context.Context, _ tokens.RoxClaims, _ ...tokens.Option) (*tokens.TokenInfo, error) {
	return nil, nil
}

type fakeRoleMapper struct {
	ID string
}

func (m *fakeRoleMapper) FromUserDescriptor(_ context.Context, _ *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	return nil, nil
}

// endregion test utilities
