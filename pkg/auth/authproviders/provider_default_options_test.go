package authproviders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	tokenIssuerMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDefaultNewID(t *testing.T) {
	option := DefaultNewID()

	// DefaultNewID should fail on a provider object with nil storedInfo field.
	testNoStoredInfoProvider(t, option, noStoredInfoErr, extractID)

	// DefaultNewID should not overwrite a pre-existing ID,
	// nor should the associated reveret action.
	previousIDProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Id: authProviderID,
		},
	}
	testNoOverwriteOptionApplication(t, option, previousIDProvider, noIssuer, noMapper, noBackend, noBackendFactory)

	// DefaultNewID should set the ID field in the storedInfo field
	// with a valid UUID. The revert action should set the ID back
	// to its previous value (here empty string).
	provider := &providerImpl{
		storedInfo: &storage.AuthProvider{},
	}
	assert.Empty(t, provider.storedInfo.GetId())
	t.Run(emptyInfoProviderCaseName, func(it *testing.T) {
		revert, err := option(provider)
		assert.NoError(t, err)
		providerID := provider.storedInfo.GetId()
		assert.NotEmpty(t, providerID)
		_, err = uuid.FromString(providerID)
		assert.NoError(t, err)
		err = revert(provider)
		assert.NoError(t, err)
		assert.Empty(t, provider.storedInfo.GetId())
	})
}

func TestDefaultLoginURL(t *testing.T) {
	called := 0
	getLoginURL := func(_ string) string {
		called += 1
		return testLoginURL
	}
	option := DefaultLoginURL(getLoginURL)

	// DefaultLoginURL should fail on a provider object with nil storedInfo field.
	testNoStoredInfoProvider(t, option, noStoredInfoErr, extractLoginURL)
	// Ensure the option application did not call getLoginURL.
	assert.Zero(t, called)

	// DefaultLoginURL should not overwrite a pre-existing login URL,
	// nor should the revert action.
	previousLoginURLProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			LoginUrl: baseLoginURL,
		},
	}
	testNoOverwriteOptionApplication(t, option, previousLoginURLProvider, noIssuer, noMapper, noBackend, noBackendFactory)
	// Ensure the option application did not call getLoginURL.
	assert.Zero(t, called)

	// DefaultLoginURL on a provider with non-nil stored info but no
	// previous login URL should set the login URL to the result of the
	// provided function call, the revert action should reset
	// the login URL to an empty value.
	provider := &providerImpl{
		storedInfo: &storage.AuthProvider{},
	}
	assert.Empty(t, extractLoginURL(provider))
	t.Run(
		emptyInfoProviderCaseName,
		testProviderOptionApplication(
			option,
			&storage.AuthProvider{},
			testLoginURL,
			extractLoginURL,
		),
	)
	// ensure the getLoginURL function was called while applying the option.
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
	testNoOverwriteOptionApplication(t, option, previousIssuerProvider, baseIssuer, noMapper, noBackend, noBackendFactory)

	// DefaultIssuerFromFactory should set the issuer to the result
	// of the provider factory creation call, the revert action should
	// reset the issuer to a nil value.
	noIssuerProvider := &providerImpl{}
	assert.Nil(t, noIssuerProvider.issuer)
	t.Run("Provider with no previous token issuer", func(it *testing.T) {
		revert, err := option(noIssuerProvider)
		assert.NoError(it, err)
		assert.Equal(it, testIssuer, noIssuerProvider.issuer)
		err = revert(noIssuerProvider)
		assert.NoError(it, err)
		assert.Nil(it, noIssuerProvider.issuer)
	})
}

func TestDefaultRoleMapperOption(t *testing.T) {
	optionBaseRoleMapper := &fakeRoleMapper{ID: baseRoleMapperID}
	optionTestRoleMapper := &fakeRoleMapper{ID: testRoleMapperID}

	getRoleMapper := func(_ string) permissions.RoleMapper {
		return optionTestRoleMapper
	}
	option := DefaultRoleMapperOption(getRoleMapper)

	// DefaultRoleMapper should not erase any pre-existing role mapper,
	// nor should the associated revert action.
	previousRoleMapperProvider := &providerImpl{
		roleMapper: optionBaseRoleMapper,
	}
	testNoOverwriteOptionApplication(t, option, previousRoleMapperProvider, noIssuer, optionBaseRoleMapper, noBackend, noBackendFactory)

	// DefaultRoleMapper should not touch the role mapper if there is
	// no previous role mapper, but the storage can NOT provide an ID,
	// the associated revert action should not touch the role mapper either.
	t.Run("Provider with no previous role mapper nor ID", func(it *testing.T) {
		noIDNoMapperProvider := &providerImpl{}
		assert.Nil(it, noIDNoMapperProvider.roleMapper)
		assert.Empty(it, extractID(noIDNoMapperProvider))
		revert, err := option(noIDNoMapperProvider)
		assert.NoError(it, err)
		assert.Nil(it, noIDNoMapperProvider.roleMapper)
		err = revert(noIDNoMapperProvider)
		assert.NoError(it, err)
		assert.Nil(it, noIDNoMapperProvider.roleMapper)
	})

	// DefaultRoleMapper should set the role mapper if there is
	// no previous mapper and the storage view can provide an ID.
	// The revert action should reset the role mapper to a nil value.
	t.Run("Provider with ID but no previous role mapper", func(it *testing.T) {
		provider := &providerImpl{
			storedInfo: &storage.AuthProvider{
				Id: authProviderID,
			},
		}
		assert.Nil(it, provider.roleMapper)
		assert.Equal(it, authProviderID, extractID(provider))
		revert, err := option(provider)
		assert.NoError(it, err)
		assert.Equal(it, optionTestRoleMapper, provider.roleMapper)
		err = revert(provider)
		assert.NoError(it, err)
		assert.Nil(it, err)
	})
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
	testNoOverwriteOptionApplication(t, option, previousBackendProvider, noIssuer, noMapper, baseAuthProviderBackend, baseAuthProviderBackendFactory)

	t.Run("Provider with wrong type", func(it *testing.T) {
		provider := &providerImpl{
			storedInfo: &storage.AuthProvider{
				Type: "base",
			},
		}
		assert.Nil(it, provider.backend)
		assert.Nil(it, provider.backendFactory)
		assert.Empty(it, provider.storedInfo.GetConfig())
		revert, err := option(provider)
		assert.Error(t, err)
		assert.Nil(it, provider.backend)
		assert.Nil(it, provider.backendFactory)
		assert.Empty(it, provider.storedInfo.GetConfig())
		err = revert(provider)
		assert.NoError(it, err)
		assert.Nil(it, provider.backend)
		assert.Nil(it, provider.backendFactory)
		assert.Empty(it, provider.storedInfo.GetConfig())
	})

	t.Run("Provider without backend", func(it *testing.T) {
		provider := &providerImpl{
			backendFactory: baseAuthProviderBackendFactory,
			storedInfo: &storage.AuthProvider{
				Type:   "test",
				Config: baseAuthProviderBackendConfig,
			},
		}
		assert.Nil(it, provider.backend)
		assert.Equal(it, baseAuthProviderBackendFactory, provider.backendFactory)
		assert.Equal(it, baseAuthProviderBackendConfig, extractConfig(provider))
		revert, err := option(provider)
		assert.NoError(it, err)
		assert.Equal(it, testAuthProviderBackend, provider.backend)
		assert.Equal(it, testAuthProviderBackendFactory, provider.backendFactory)
		assert.Equal(it, testAuthProviderBackendConfig, extractConfig(provider))
		err = revert(provider)
		assert.NoError(it, err)
		assert.Nil(it, provider.backend)
		assert.Equal(it, baseAuthProviderBackendFactory, provider.backendFactory)
		assert.Equal(it, baseAuthProviderBackendConfig, extractConfig(provider))
	})
}

// region test helpers

func testNoOverwriteOptionApplication(
	t *testing.T,
	option ProviderOption,
	provider *providerImpl,
	oldTokenIssuer tokens.Issuer,
	oldRoleMapper permissions.RoleMapper,
	oldBackend Backend,
	oldBackendFactory BackendFactory,
) {
	t.Run(noOverwriteCaseName, func(it *testing.T) {
		oldStoredInfo := provider.storedInfo.CloneVT()
		revert, err := option(provider)
		assert.NoError(it, err)
		protoassert.Equal(it, oldStoredInfo, provider.storedInfo)
		assert.Equal(it, oldTokenIssuer, provider.issuer)
		assert.Equal(it, oldRoleMapper, provider.roleMapper)
		assert.Equal(it, oldBackend, provider.backend)
		assert.Equal(it, oldBackendFactory, provider.backendFactory)
		err = revert(provider)
		assert.NoError(it, err)
		protoassert.Equal(it, oldStoredInfo, provider.storedInfo)
		assert.Equal(it, oldTokenIssuer, provider.issuer)
		assert.Equal(it, oldRoleMapper, provider.roleMapper)
		assert.Equal(it, oldBackend, provider.backend)
		assert.Equal(it, oldBackendFactory, provider.backendFactory)
	})
}

// endregion test helpers
