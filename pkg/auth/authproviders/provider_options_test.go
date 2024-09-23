package authproviders

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	tokenIssuerMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	authProviderID = "41757468-5072-6f76-6964-657200111111"
	testProviderID = "41757468-5072-6f76-6964-657200222222"

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

	baseAuthProviderType = "base"
	testAuthProviderType = "test"

	baseAuthProviderName = "Auth Provider"
	testAuthProviderName = "Test Auth Provider"
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

func TestWithBackendFromFactory(t *testing.T) {
	option := WithBackendFromFactory(context.Background(), testAuthProviderBackendFactory)

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
	assert.Equal(t, testAuthProviderBackend, previousBackendProvider.backend)
	assert.Equal(t, testAuthProviderBackendFactory, previousBackendProvider.backendFactory)
	assert.Equal(t, testAuthProviderBackendConfig, previousBackendProvider.storedInfo.GetConfig())
	err = revert1(previousBackendProvider)
	assert.NoError(t, err)
	assert.Equal(t, baseAuthProviderBackend, previousBackendProvider.backend)
	assert.Equal(t, baseAuthProviderBackendFactory, previousBackendProvider.backendFactory)
	assert.Equal(t, baseAuthProviderBackendConfig, previousBackendProvider.storedInfo.GetConfig())

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

func TestDoNotStore(t *testing.T) {
	option := DoNotStore()

	baseProvider := &providerImpl{}
	assert.False(t, baseProvider.doNotStore)
	revert1, err := option(baseProvider)
	assert.NoError(t, err)
	assert.True(t, baseProvider.doNotStore)
	err = revert1(baseProvider)
	assert.NoError(t, err)
	assert.False(t, baseProvider.doNotStore)

	doNotStoreProvider := &providerImpl{
		doNotStore: true,
	}
	assert.True(t, doNotStoreProvider.doNotStore)
	revert2, err := option(doNotStoreProvider)
	assert.NoError(t, err)
	assert.True(t, doNotStoreProvider.doNotStore)
	err = revert2(doNotStoreProvider)
	assert.NoError(t, err)
	assert.True(t, doNotStoreProvider.doNotStore)

	doStoreProvider := &providerImpl{
		doNotStore: false,
	}
	assert.False(t, doStoreProvider.doNotStore)
	revert3, err := option(doStoreProvider)
	assert.NoError(t, err)
	assert.True(t, doStoreProvider.doNotStore)
	err = revert3(doStoreProvider)
	assert.NoError(t, err)
	assert.False(t, doStoreProvider.doNotStore)
}

func TestWithRoleMapper(t *testing.T) {
	baseRoleMapper := &fakeRoleMapper{ID: baseRoleMapperID}
	optionTestRoleMapper := &fakeRoleMapper{ID: testRoleMapperID}

	option := WithRoleMapper(optionTestRoleMapper)

	previousMapperProvider := &providerImpl{
		roleMapper: baseRoleMapper,
	}
	assert.Equal(t, baseRoleMapper, previousMapperProvider.roleMapper)
	revert1, err := option(previousMapperProvider)
	assert.NoError(t, err)
	assert.Equal(t, optionTestRoleMapper, previousMapperProvider.roleMapper)
	err = revert1(previousMapperProvider)
	assert.NoError(t, err)
	assert.Equal(t, baseRoleMapper, previousMapperProvider.roleMapper)

	noIDProvider := &providerImpl{}
	assert.Nil(t, noIDProvider.roleMapper)
	assert.Empty(t, noIDProvider.storedInfo.GetId())
	revert2, err := option(noIDProvider)
	assert.NoError(t, err)
	assert.Equal(t, optionTestRoleMapper, noIDProvider.roleMapper)
	err = revert2(noIDProvider)
	assert.NoError(t, err)
	assert.Nil(t, noIDProvider.roleMapper)

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

func TestWithStorageView(t *testing.T) {
	baseStorageView := &storage.AuthProvider{
		Id: authProviderID,
	}
	testStorageView := &storage.AuthProvider{
		Id: testProviderID,
	}
	option := WithStorageView(testStorageView)

	noStorageViewProvider := &providerImpl{}
	assert.Nil(t, noStorageViewProvider.storedInfo)
	revert1, err := option(noStorageViewProvider)
	assert.NoError(t, err)
	protoassert.Equal(t, testStorageView, noStorageViewProvider.storedInfo)
	err = revert1(noStorageViewProvider)
	assert.NoError(t, err)
	assert.Nil(t, noStorageViewProvider.storedInfo)

	previousStorageViewProvider := &providerImpl{
		storedInfo: baseStorageView,
	}
	protoassert.Equal(t, baseStorageView, previousStorageViewProvider.storedInfo)
	revert2, err := option(previousStorageViewProvider)
	assert.NoError(t, err)
	protoassert.Equal(t, testStorageView, previousStorageViewProvider.storedInfo)
	// validate that the provider stored info is a copy of the input object
	testStorageView.Type = "changed"
	protoassert.NotEqual(t, testStorageView, previousStorageViewProvider.storedInfo)
	err = revert2(previousStorageViewProvider)
	assert.NoError(t, err)
	protoassert.Equal(t, baseStorageView, previousStorageViewProvider.storedInfo)
}

func TestWithID(t *testing.T) {
	option := WithID(testProviderID)

	noStorageProvider := &providerImpl{}
	assert.Nil(t, noStorageProvider.storedInfo)
	assert.Empty(t, noStorageProvider.storedInfo.GetId())
	revert1, err := option(noStorageProvider)
	assert.ErrorIs(t, err, errox.InvariantViolation)
	assert.Nil(t, noStorageProvider.storedInfo)
	assert.Empty(t, noStorageProvider.storedInfo.GetId())
	err = revert1(noStorageProvider)
	assert.NoError(t, err)
	assert.Nil(t, noStorageProvider.storedInfo)
	assert.Empty(t, noStorageProvider.storedInfo.GetId())

	noIDProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{},
	}
	assert.NotNil(t, noIDProvider.storedInfo)
	assert.Empty(t, noIDProvider.storedInfo.GetId())
	revert2, err := option(noIDProvider)
	assert.NoError(t, err)
	assert.NotNil(t, noIDProvider.storedInfo)
	assert.Equal(t, testProviderID, noIDProvider.storedInfo.GetId())
	err = revert2(noIDProvider)
	assert.NoError(t, err)
	assert.NotNil(t, noIDProvider.storedInfo)
	assert.Empty(t, noIDProvider.storedInfo.GetId())

	prevIDProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Id: authProviderID,
		},
	}
	assert.NotNil(t, prevIDProvider.storedInfo)
	assert.Equal(t, authProviderID, prevIDProvider.storedInfo.GetId())
	revert3, err := option(prevIDProvider)
	assert.NoError(t, err)
	assert.NotNil(t, prevIDProvider.storedInfo)
	assert.Equal(t, testProviderID, prevIDProvider.storedInfo.GetId())
	err = revert3(prevIDProvider)
	assert.NoError(t, err)
	assert.NotNil(t, prevIDProvider.storedInfo)
	assert.Equal(t, authProviderID, prevIDProvider.storedInfo.GetId())

	breakRevertProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Id: authProviderID,
		},
	}
	assert.NotNil(t, breakRevertProvider.storedInfo)
	assert.Equal(t, authProviderID, breakRevertProvider.storedInfo.GetId())
	revert4, err := option(breakRevertProvider)
	assert.NoError(t, err)
	assert.NotNil(t, breakRevertProvider.storedInfo)
	assert.Equal(t, testProviderID, breakRevertProvider.storedInfo.GetId())
	breakRevertProvider.storedInfo = nil
	assert.Nil(t, breakRevertProvider.storedInfo)
	assert.Empty(t, breakRevertProvider.storedInfo.GetId())
	err = revert4(breakRevertProvider)
	assert.ErrorIs(t, err, errox.InvariantViolation)
	assert.Nil(t, breakRevertProvider.storedInfo)
	assert.Empty(t, breakRevertProvider.storedInfo.GetId())
}

func TestWithType(t *testing.T) {
	option := WithType(testAuthProviderType)

	noStorageProvider := &providerImpl{}
	assert.Nil(t, noStorageProvider.storedInfo)
	assert.Empty(t, noStorageProvider.storedInfo.GetType())
	revert1, err := option(noStorageProvider)
	assert.ErrorIs(t, err, errox.InvariantViolation)
	assert.Nil(t, noStorageProvider.storedInfo)
	assert.Empty(t, noStorageProvider.storedInfo.GetType())
	err = revert1(noStorageProvider)
	assert.NoError(t, err)
	assert.Nil(t, noStorageProvider.storedInfo)
	assert.Empty(t, noStorageProvider.storedInfo.GetType())

	noTypeProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{},
	}
	assert.NotNil(t, noTypeProvider.storedInfo)
	assert.Empty(t, noTypeProvider.storedInfo.GetType())
	revert2, err := option(noTypeProvider)
	assert.NoError(t, err)
	assert.NotNil(t, noTypeProvider.storedInfo)
	assert.Equal(t, testAuthProviderType, noTypeProvider.storedInfo.GetType())
	err = revert2(noTypeProvider)
	assert.NoError(t, err)
	assert.NotNil(t, noTypeProvider.storedInfo)
	assert.Empty(t, noTypeProvider.storedInfo.GetType())

	prevTypeProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Type: baseAuthProviderType,
		},
	}
	assert.NotNil(t, prevTypeProvider.storedInfo)
	assert.Equal(t, baseAuthProviderType, prevTypeProvider.storedInfo.GetType())
	revert3, err := option(prevTypeProvider)
	assert.NoError(t, err)
	assert.NotNil(t, prevTypeProvider.storedInfo)
	assert.Equal(t, testAuthProviderType, prevTypeProvider.storedInfo.GetType())
	err = revert3(prevTypeProvider)
	assert.NoError(t, err)
	assert.NotNil(t, prevTypeProvider.storedInfo)
	assert.Equal(t, baseAuthProviderType, prevTypeProvider.storedInfo.GetType())

	breakRevertProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Type: baseAuthProviderType,
		},
	}
	assert.NotNil(t, breakRevertProvider.storedInfo)
	assert.Equal(t, baseAuthProviderType, breakRevertProvider.storedInfo.GetType())
	revert4, err := option(breakRevertProvider)
	assert.NoError(t, err)
	assert.NotNil(t, breakRevertProvider.storedInfo)
	assert.Equal(t, testAuthProviderType, breakRevertProvider.storedInfo.GetType())
	breakRevertProvider.storedInfo = nil
	assert.Nil(t, breakRevertProvider.storedInfo)
	assert.Empty(t, breakRevertProvider.storedInfo.GetType())
	err = revert4(breakRevertProvider)
	assert.ErrorIs(t, err, errox.InvariantViolation)
	assert.Nil(t, breakRevertProvider.storedInfo)
	assert.Empty(t, breakRevertProvider.storedInfo.GetType())
}

func TestWithName(t *testing.T) {
	option := WithName(testAuthProviderName)

	noStorageProvider := &providerImpl{}
	assert.Nil(t, noStorageProvider.storedInfo)
	assert.Empty(t, noStorageProvider.storedInfo.GetName())
	revert1, err := option(noStorageProvider)
	assert.ErrorIs(t, err, errox.InvariantViolation)
	assert.Nil(t, noStorageProvider.storedInfo)
	assert.Empty(t, noStorageProvider.storedInfo.GetName())
	err = revert1(noStorageProvider)
	assert.NoError(t, err)
	assert.Nil(t, noStorageProvider.storedInfo)
	assert.Empty(t, noStorageProvider.storedInfo.GetName())

	noTypeProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{},
	}
	assert.NotNil(t, noTypeProvider.storedInfo)
	assert.Empty(t, noTypeProvider.storedInfo.GetName())
	revert2, err := option(noTypeProvider)
	assert.NoError(t, err)
	assert.NotNil(t, noTypeProvider.storedInfo)
	assert.Equal(t, testAuthProviderName, noTypeProvider.storedInfo.GetName())
	err = revert2(noTypeProvider)
	assert.NoError(t, err)
	assert.NotNil(t, noTypeProvider.storedInfo)
	assert.Empty(t, noTypeProvider.storedInfo.GetName())

	prevNameProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Name: baseAuthProviderName,
		},
	}
	assert.NotNil(t, prevNameProvider.storedInfo)
	assert.Equal(t, baseAuthProviderName, prevNameProvider.storedInfo.GetName())
	revert3, err := option(prevNameProvider)
	assert.NoError(t, err)
	assert.NotNil(t, prevNameProvider.storedInfo)
	assert.Equal(t, testAuthProviderName, prevNameProvider.storedInfo.GetName())
	err = revert3(prevNameProvider)
	assert.NoError(t, err)
	assert.NotNil(t, prevNameProvider.storedInfo)
	assert.Equal(t, baseAuthProviderName, prevNameProvider.storedInfo.GetName())

	breakRevertProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Name: baseAuthProviderName,
		},
	}
	assert.NotNil(t, breakRevertProvider.storedInfo)
	assert.Equal(t, baseAuthProviderName, breakRevertProvider.storedInfo.GetName())
	revert4, err := option(breakRevertProvider)
	assert.NoError(t, err)
	assert.NotNil(t, breakRevertProvider.storedInfo)
	assert.Equal(t, testAuthProviderName, breakRevertProvider.storedInfo.GetName())
	breakRevertProvider.storedInfo = nil
	assert.Nil(t, breakRevertProvider.storedInfo)
	assert.Empty(t, breakRevertProvider.storedInfo.GetName())
	err = revert4(breakRevertProvider)
	assert.ErrorIs(t, err, errox.InvariantViolation)
	assert.Nil(t, breakRevertProvider.storedInfo)
	assert.Empty(t, breakRevertProvider.storedInfo.GetName())
}

func TestWithEnabled(t *testing.T) {
	doEnable := WithEnabled(true)
	doDisable := WithEnabled(false)

	noStoreViewProvider := &providerImpl{}
	assert.Nil(t, noStoreViewProvider.storedInfo)
	assert.False(t, noStoreViewProvider.storedInfo.GetEnabled())

	revert1, err := doEnable(noStoreViewProvider)
	assert.ErrorIs(t, err, errox.InvariantViolation)
	assert.Nil(t, noStoreViewProvider.storedInfo)
	assert.False(t, noStoreViewProvider.storedInfo.GetEnabled())
	err = revert1(noStoreViewProvider)
	assert.NoError(t, err)
	assert.Nil(t, noStoreViewProvider.storedInfo)
	assert.False(t, noStoreViewProvider.storedInfo.GetEnabled())

	revert2, err := doDisable(noStoreViewProvider)
	assert.ErrorIs(t, err, errox.InvariantViolation)
	assert.Nil(t, noStoreViewProvider.storedInfo)
	assert.False(t, noStoreViewProvider.storedInfo.GetEnabled())
	err = revert2(noStoreViewProvider)
	assert.Nil(t, noStoreViewProvider.storedInfo)
	assert.False(t, noStoreViewProvider.storedInfo.GetEnabled())

	defaultStoredInfoProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{},
	}
	assert.NotNil(t, defaultStoredInfoProvider.storedInfo)
	assert.False(t, defaultStoredInfoProvider.storedInfo.GetEnabled())

	revert3, err := doEnable(defaultStoredInfoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, defaultStoredInfoProvider.storedInfo)
	assert.True(t, defaultStoredInfoProvider.storedInfo.GetEnabled())
	err = revert3(defaultStoredInfoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, defaultStoredInfoProvider.storedInfo)
	assert.False(t, defaultStoredInfoProvider.storedInfo.GetEnabled())

	revert4, err := doDisable(defaultStoredInfoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, defaultStoredInfoProvider.storedInfo)
	assert.False(t, defaultStoredInfoProvider.storedInfo.GetEnabled())
	err = revert4(defaultStoredInfoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, defaultStoredInfoProvider.storedInfo)
	assert.False(t, defaultStoredInfoProvider.storedInfo.GetEnabled())

	enabledProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Enabled: true,
		},
	}
	assert.NotNil(t, enabledProvider.storedInfo)
	assert.True(t, enabledProvider.storedInfo.GetEnabled())

	revert5, err := doEnable(enabledProvider)
	assert.NoError(t, err)
	assert.NotNil(t, enabledProvider.storedInfo)
	assert.True(t, enabledProvider.storedInfo.GetEnabled())
	err = revert5(enabledProvider)
	assert.NoError(t, err)
	assert.NotNil(t, enabledProvider.storedInfo)
	assert.True(t, enabledProvider.storedInfo.GetEnabled())

	revert6, err := doDisable(enabledProvider)
	assert.NoError(t, err)
	assert.NotNil(t, enabledProvider.storedInfo)
	assert.False(t, enabledProvider.storedInfo.GetEnabled())
	err = revert6(enabledProvider)
	assert.NoError(t, err)
	assert.NotNil(t, enabledProvider.storedInfo)
	assert.True(t, enabledProvider.storedInfo.GetEnabled())

	disabledProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Enabled: false,
		},
	}
	assert.NotNil(t, disabledProvider.storedInfo)
	assert.False(t, disabledProvider.storedInfo.GetEnabled())

	revert7, err := doEnable(disabledProvider)
	assert.NoError(t, err)
	assert.NotNil(t, disabledProvider.storedInfo)
	assert.True(t, disabledProvider.storedInfo.GetEnabled())
	err = revert7(disabledProvider)
	assert.NoError(t, err)
	assert.NotNil(t, disabledProvider.storedInfo)
	assert.False(t, disabledProvider.storedInfo.GetEnabled())

	revert8, err := doDisable(disabledProvider)
	assert.NoError(t, err)
	assert.NotNil(t, disabledProvider.storedInfo)
	assert.False(t, disabledProvider.storedInfo.GetEnabled())
	err = revert8(disabledProvider)
	assert.NoError(t, err)
	assert.NotNil(t, disabledProvider.storedInfo)
	assert.False(t, disabledProvider.storedInfo.GetEnabled())

	breakEnableRevertProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Enabled: false,
		},
	}
	assert.NotNil(t, breakEnableRevertProvider.storedInfo)
	assert.False(t, breakEnableRevertProvider.storedInfo.GetEnabled())
	revert9, err := doEnable(breakEnableRevertProvider)
	assert.NoError(t, err)
	assert.NotNil(t, breakEnableRevertProvider.storedInfo)
	assert.True(t, breakEnableRevertProvider.storedInfo.GetEnabled())
	breakEnableRevertProvider.storedInfo = nil
	assert.Nil(t, breakEnableRevertProvider.storedInfo)
	assert.False(t, breakEnableRevertProvider.storedInfo.GetEnabled())
	err = revert9(breakEnableRevertProvider)
	assert.ErrorIs(t, err, errox.InvariantViolation)
	assert.Nil(t, breakEnableRevertProvider.storedInfo)
	assert.False(t, breakEnableRevertProvider.storedInfo.GetEnabled())

	breakDisableRevertProvider := &providerImpl{
		storedInfo: &storage.AuthProvider{
			Enabled: true,
		},
	}
	assert.NotNil(t, breakDisableRevertProvider.storedInfo)
	assert.True(t, breakDisableRevertProvider.storedInfo.GetEnabled())
	revert10, err := doDisable(breakDisableRevertProvider)
	assert.NoError(t, err)
	assert.NotNil(t, breakDisableRevertProvider.storedInfo)
	assert.False(t, breakDisableRevertProvider.storedInfo.GetEnabled())
	breakDisableRevertProvider.storedInfo = nil
	assert.Nil(t, breakDisableRevertProvider.storedInfo)
	assert.False(t, breakDisableRevertProvider.storedInfo.GetEnabled())
	err = revert10(breakDisableRevertProvider)
	assert.ErrorIs(t, err, errox.InvariantViolation)
	assert.Nil(t, breakDisableRevertProvider.storedInfo)
	assert.False(t, breakDisableRevertProvider.storedInfo.GetEnabled())

}

func TestWithActive(t *testing.T) {

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
