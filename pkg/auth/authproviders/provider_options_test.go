package authproviders

import (
	"context"
	"fmt"
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

	testCases := map[string]struct {
		provider     *providerImpl
		initialState bool
	}{
		"Base provider": {
			provider:     &providerImpl{},
			initialState: false,
		},
		"Do not store provider": {
			provider: &providerImpl{
				doNotStore: true,
			},
			initialState: true,
		},
		"Do store provider": {
			provider: &providerImpl{
				doNotStore: false,
			},
			initialState: false,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(it *testing.T) {
			provider := tc.provider
			assert.Equal(it, tc.initialState, provider.doNotStore)
			revert, err := option(provider)
			assert.NoError(it, err)
			assert.True(it, provider.doNotStore)
			err = revert(provider)
			assert.NoError(it, err)
			assert.Equal(it, tc.initialState, provider.doNotStore)
		})
	}
}

func TestWithRoleMapper(t *testing.T) {
	baseRoleMapper := &fakeRoleMapper{ID: baseRoleMapperID}
	optionTestRoleMapper := &fakeRoleMapper{ID: testRoleMapperID}

	option := WithRoleMapper(optionTestRoleMapper)

	testCases := map[string]struct {
		provider          *providerImpl
		providerID        string
		initialRoleMapper permissions.RoleMapper
	}{
		"Provider with previous role mapper": {
			provider: &providerImpl{
				roleMapper: baseRoleMapper,
			},
			initialRoleMapper: baseRoleMapper,
		},
		"Provider with no stored ID and no role mapper": {
			provider: &providerImpl{},
		},
		"Provider with stored ID but no role mapper": {
			provider: &providerImpl{
				storedInfo: &storage.AuthProvider{
					Id: authProviderID,
				},
			},
			providerID: authProviderID,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(it *testing.T) {
			provider := tc.provider
			if tc.initialRoleMapper == nil {
				assert.Nil(it, provider.roleMapper)
			} else {
				assert.Equal(it, tc.initialRoleMapper, provider.roleMapper)
			}
			assert.Equal(it, tc.providerID, provider.storedInfo.GetId())
			revert, err := option(provider)
			assert.NoError(it, err)
			assert.Equal(it, optionTestRoleMapper, provider.roleMapper)
			err = revert(provider)
			assert.NoError(it, err)
			if tc.initialRoleMapper == nil {
				assert.Nil(it, provider.roleMapper)
			} else {
				assert.Equal(it, tc.initialRoleMapper, provider.roleMapper)
			}
		})
	}
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

	t.Run("Provider without storedInfo", func(it *testing.T) {
		noStorageProvider := &providerImpl{}
		assert.Nil(it, noStorageProvider.storedInfo)
		assert.Empty(it, noStorageProvider.storedInfo.GetId())
		revert1, err := option(noStorageProvider)
		assert.ErrorIs(it, err, errox.InvariantViolation)
		assert.Nil(it, noStorageProvider.storedInfo)
		assert.Empty(it, noStorageProvider.storedInfo.GetId())
		err = revert1(noStorageProvider)
		assert.NoError(it, err)
		assert.Nil(it, noStorageProvider.storedInfo)
		assert.Empty(it, noStorageProvider.storedInfo.GetId())
	})

	testCases := map[string]struct {
		storedInfo *storage.AuthProvider
		providerID string
	}{
		"Provider with storedInfo but no ID": {
			storedInfo: &storage.AuthProvider{},
		},
		"Provider with storedInfo and previous ID": {
			storedInfo: &storage.AuthProvider{
				Id: authProviderID,
			},
			providerID: authProviderID,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(it *testing.T) {
			provider := &providerImpl{storedInfo: tc.storedInfo}
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.providerID, provider.storedInfo.GetId())
			revert, err := option(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, testProviderID, provider.storedInfo.GetId())
			err = revert(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.providerID, provider.storedInfo.GetId())
		})
		t.Run(name+" - breaking revert", func(it *testing.T) {
			provider := &providerImpl{storedInfo: tc.storedInfo}
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.providerID, provider.storedInfo.GetId())
			revert, err := option(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, testProviderID, provider.storedInfo.GetId())
			provider.storedInfo = nil
			assert.Nil(it, provider.storedInfo)
			assert.Empty(it, provider.storedInfo.GetId())
			err = revert(provider)
			assert.ErrorIs(it, err, errox.InvariantViolation)
			assert.Nil(it, provider.storedInfo)
			assert.Empty(it, provider.storedInfo.GetId())
		})
	}
}

func TestWithType(t *testing.T) {
	option := WithType(testAuthProviderType)

	t.Run("Provider without storedInfo", func(it *testing.T) {
		noStorageProvider := &providerImpl{}
		assert.Nil(it, noStorageProvider.storedInfo)
		assert.Empty(it, noStorageProvider.storedInfo.GetType())
		revert1, err := option(noStorageProvider)
		assert.ErrorIs(it, err, errox.InvariantViolation)
		assert.Nil(it, noStorageProvider.storedInfo)
		assert.Empty(it, noStorageProvider.storedInfo.GetType())
		err = revert1(noStorageProvider)
		assert.NoError(it, err)
		assert.Nil(it, noStorageProvider.storedInfo)
		assert.Empty(it, noStorageProvider.storedInfo.GetType())
	})

	testCases := map[string]struct {
		storedInfo   *storage.AuthProvider
		providerType string
	}{
		"Provider with stored info but no type": {
			storedInfo:   &storage.AuthProvider{},
			providerType: "",
		},
		"Provider with stored info and previous type": {
			storedInfo: &storage.AuthProvider{
				Type: baseAuthProviderType,
			},
			providerType: baseAuthProviderType,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(it *testing.T) {
			provider := &providerImpl{storedInfo: tc.storedInfo}
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.providerType, provider.storedInfo.GetType())
			revert, err := option(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, testAuthProviderType, provider.storedInfo.GetType())
			err = revert(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.providerType, provider.storedInfo.GetType())
		})
		t.Run(name+" - breaking revert", func(it *testing.T) {
			provider := &providerImpl{storedInfo: tc.storedInfo}
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.providerType, provider.storedInfo.GetType())
			revert, err := option(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, testAuthProviderType, provider.storedInfo.GetType())
			provider.storedInfo = nil
			assert.Nil(it, provider.storedInfo)
			assert.Empty(it, provider.storedInfo.GetType())
			err = revert(provider)
			assert.ErrorIs(it, err, errox.InvariantViolation)
			assert.Nil(it, provider.storedInfo)
			assert.Empty(it, provider.storedInfo.GetType())
		})
	}
}

func TestWithName(t *testing.T) {
	option := WithName(testAuthProviderName)

	t.Run("Provider without storedInfo", func(it *testing.T) {
		noStorageProvider := &providerImpl{}
		assert.Nil(it, noStorageProvider.storedInfo)
		assert.Empty(it, noStorageProvider.storedInfo.GetName())
		revert1, err := option(noStorageProvider)
		assert.ErrorIs(it, err, errox.InvariantViolation)
		assert.Nil(it, noStorageProvider.storedInfo)
		assert.Empty(it, noStorageProvider.storedInfo.GetName())
		err = revert1(noStorageProvider)
		assert.NoError(it, err)
		assert.Nil(it, noStorageProvider.storedInfo)
		assert.Empty(it, noStorageProvider.storedInfo.GetName())
	})

	testCases := map[string]struct {
		storedInfo   *storage.AuthProvider
		providerName string
	}{
		"Provider with stored info but no name": {
			storedInfo:   &storage.AuthProvider{},
			providerName: "",
		},
		"Provider with stored info and previous name": {
			storedInfo: &storage.AuthProvider{
				Name: baseAuthProviderName,
			},
			providerName: baseAuthProviderName,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(it *testing.T) {
			provider := &providerImpl{storedInfo: tc.storedInfo}
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.providerName, provider.storedInfo.GetName())
			revert, err := option(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, testAuthProviderName, provider.storedInfo.GetName())
			err = revert(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.providerName, provider.storedInfo.GetName())
		})
		t.Run(name+" - breaking revert", func(it *testing.T) {
			provider := &providerImpl{storedInfo: tc.storedInfo}
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.providerName, provider.storedInfo.GetName())
			revert, err := option(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, testAuthProviderName, provider.storedInfo.GetName())
			provider.storedInfo = nil
			assert.Nil(it, provider.storedInfo)
			assert.Empty(it, provider.storedInfo.GetName())
			err = revert(provider)
			assert.ErrorIs(it, err, errox.InvariantViolation)
			assert.Nil(it, provider.storedInfo)
			assert.Empty(it, provider.storedInfo.GetName())
		})
	}
}

func TestWithEnabled(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("Provider with no stored info, setting enabled to %t", enabled), func(it *testing.T) {
			provider := &providerImpl{}
			option := WithEnabled(enabled)
			assert.Nil(it, provider.storedInfo)
			assert.False(it, provider.storedInfo.GetEnabled())
			revert, err := option(provider)
			assert.ErrorIs(it, err, errox.InvariantViolation)
			assert.Nil(it, provider.storedInfo)
			assert.False(it, provider.storedInfo.GetEnabled())
			err = revert(provider)
			assert.NoError(it, err)
			assert.Nil(it, provider.storedInfo)
			assert.False(it, provider.storedInfo.GetEnabled())
		})
	}

	testCases := map[string]struct {
		storedInfo     *storage.AuthProvider
		initialEnabled bool
		targetEnabled  bool
	}{
		"Provider with stored information but enable not set - enabling": {
			storedInfo:     &storage.AuthProvider{},
			initialEnabled: false,
			targetEnabled:  true,
		},
		"Provider with stored information but enable not set - disabling": {
			storedInfo:     &storage.AuthProvider{},
			initialEnabled: false,
			targetEnabled:  false,
		},
		"Provider with stored information and enabled - enabling": {
			storedInfo: &storage.AuthProvider{
				Enabled: true,
			},
			initialEnabled: true,
			targetEnabled:  true,
		},
		"Provider with stored information and enabled - disabling": {
			storedInfo: &storage.AuthProvider{
				Enabled: true,
			},
			initialEnabled: true,
			targetEnabled:  false,
		},
		"Provider with stored information and disabled - enabling": {
			storedInfo: &storage.AuthProvider{
				Enabled: false,
			},
			initialEnabled: false,
			targetEnabled:  true,
		},
		"Provider with stored information and disabled - disabling": {
			storedInfo: &storage.AuthProvider{
				Enabled: false,
			},
			initialEnabled: false,
			targetEnabled:  false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(it *testing.T) {
			option := WithEnabled(tc.targetEnabled)
			provider := &providerImpl{storedInfo: tc.storedInfo}
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.initialEnabled, provider.storedInfo.GetEnabled())
			revert, err := option(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.targetEnabled, provider.storedInfo.GetEnabled())
			err = revert(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.initialEnabled, provider.storedInfo.GetEnabled())
		})
		t.Run(name+" - breaking revert", func(it *testing.T) {
			option := WithEnabled(tc.targetEnabled)
			provider := &providerImpl{storedInfo: tc.storedInfo}
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.initialEnabled, provider.storedInfo.GetEnabled())
			revert, err := option(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.targetEnabled, provider.storedInfo.GetEnabled())
			provider.storedInfo = nil
			assert.Nil(it, provider.storedInfo)
			assert.False(it, provider.storedInfo.GetEnabled())
			err = revert(provider)
			assert.ErrorIs(it, err, errox.InvariantViolation)
			assert.Nil(it, provider.storedInfo)
			assert.False(it, provider.storedInfo.GetEnabled())

		})
	}
}

func TestWithActive(t *testing.T) {
	for _, activate := range []bool{true, false} {
		t.Run(fmt.Sprintf("Provider with no stored info - activate to %t", activate), func(it *testing.T) {
			option := WithActive(activate)
			provider := &providerImpl{}
			assert.Nil(it, provider.storedInfo)
			assert.False(it, provider.storedInfo.GetActive())
			revert, err := option(provider)
			assert.ErrorIs(it, err, errox.InvariantViolation)
			assert.Nil(it, provider.storedInfo)
			assert.False(it, provider.storedInfo.GetActive())
			err = revert(provider)
			assert.NoError(it, err)
			assert.Nil(it, provider.storedInfo)
			assert.False(it, provider.storedInfo.GetActive())
		})
	}

	testCases := map[string]struct {
		storedInfo       *storage.AuthProvider
		initialValidated bool
		initialActive    bool
		targetActive     bool
	}{
		"Provider with stored info, but no active nor validated data, activating": {
			storedInfo:       &storage.AuthProvider{},
			initialActive:    false,
			initialValidated: false,
			targetActive:     true,
		},
		"Provider with stored info, but no active nor validated data, deactivating": {
			storedInfo:       &storage.AuthProvider{},
			initialActive:    false,
			initialValidated: false,
			targetActive:     false,
		},
		"Provider with stored info, active, but no validated data, activating": {
			storedInfo: &storage.AuthProvider{
				Active: true,
			},
			initialActive:    true,
			initialValidated: false,
			targetActive:     true,
		},
		"Provider with stored info, active, but no validated data, deactivating": {
			storedInfo: &storage.AuthProvider{
				Active: true,
			},
			initialActive:    true,
			initialValidated: false,
			targetActive:     false,
		},
		"Provider with stored info, inactive, but no validated data, activating": {
			storedInfo: &storage.AuthProvider{
				Active: false,
			},
			initialActive:    false,
			initialValidated: false,
			targetActive:     true,
		},
		"Provider with stored info, inactive, but no validated data, deactivating": {
			storedInfo: &storage.AuthProvider{
				Active: false,
			},
			initialActive:    false,
			initialValidated: false,
			targetActive:     false,
		},
		"Provider with stored info, validated, but no active data, activating": {
			storedInfo: &storage.AuthProvider{
				Validated: true,
			},
			initialActive:    false,
			initialValidated: true,
			targetActive:     true,
		},
		"Provider with stored info, validated, but no active data, deactivating": {
			storedInfo: &storage.AuthProvider{
				Validated: true,
			},
			initialActive:    false,
			initialValidated: true,
			targetActive:     false,
		},
		"Provider with stored info, not validated, no active data, activating": {
			storedInfo: &storage.AuthProvider{
				Validated: false,
			},
			initialActive:    false,
			initialValidated: false,
			targetActive:     true,
		},
		"Provider with stored info, not validated, no active data, deactivating": {
			storedInfo: &storage.AuthProvider{
				Validated: false,
			},
			initialActive:    false,
			initialValidated: false,
			targetActive:     false,
		},
		"Provider with stored info, active, validated, activating": {
			storedInfo: &storage.AuthProvider{
				Active:    true,
				Validated: true,
			},
			initialActive:    true,
			initialValidated: true,
			targetActive:     true,
		},
		"Provider with stored info, active, validated, deactivating": {
			storedInfo: &storage.AuthProvider{
				Active:    true,
				Validated: true,
			},
			initialActive:    true,
			initialValidated: true,
			targetActive:     false,
		},
		"Provider with stored info, not active, validated, activating": {
			storedInfo: &storage.AuthProvider{
				Active:    false,
				Validated: true,
			},
			initialActive:    false,
			initialValidated: true,
			targetActive:     true,
		},
		"Provider with stored info, not active, validated, deactivating": {
			storedInfo: &storage.AuthProvider{
				Active:    false,
				Validated: true,
			},
			initialActive:    false,
			initialValidated: true,
			targetActive:     false,
		},
		"Provider with stored info, active, not validated, activating": {
			storedInfo: &storage.AuthProvider{
				Active:    true,
				Validated: false,
			},
			initialActive:    true,
			initialValidated: false,
			targetActive:     true,
		},
		"Provider with stored info, active, not validated, deactivating": {
			storedInfo: &storage.AuthProvider{
				Active:    true,
				Validated: false,
			},
			initialActive:    true,
			initialValidated: false,
			targetActive:     false,
		},
		"Provider with stored info, not active, not validated, activating": {
			storedInfo: &storage.AuthProvider{
				Active:    false,
				Validated: false,
			},
			initialActive:    false,
			initialValidated: false,
			targetActive:     true,
		},
		"Provider with stored info, not active, not validated, deactivating": {
			storedInfo: &storage.AuthProvider{
				Active:    false,
				Validated: false,
			},
			initialActive:    false,
			initialValidated: false,
			targetActive:     false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(it *testing.T) {
			option := WithActive(tc.targetActive)
			provider := &providerImpl{storedInfo: tc.storedInfo}
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.initialActive, provider.storedInfo.GetActive())
			assert.Equal(it, tc.initialValidated, provider.storedInfo.GetValidated())
			revert, err := option(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.targetActive, provider.storedInfo.GetActive())
			assert.Equal(it, tc.targetActive, provider.storedInfo.GetValidated())
			err = revert(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.initialActive, provider.storedInfo.GetActive())
			assert.Equal(it, tc.initialValidated, provider.storedInfo.GetValidated())
		})
		t.Run(name+" - breaking revert", func(it *testing.T) {
			option := WithActive(tc.targetActive)
			provider := &providerImpl{storedInfo: tc.storedInfo}
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.initialActive, provider.storedInfo.GetActive())
			assert.Equal(it, tc.initialValidated, provider.storedInfo.GetValidated())
			revert, err := option(provider)
			assert.NoError(it, err)
			assert.NotNil(it, provider.storedInfo)
			assert.Equal(it, tc.targetActive, provider.storedInfo.GetActive())
			assert.Equal(it, tc.targetActive, provider.storedInfo.GetValidated())
			provider.storedInfo = nil
			assert.Nil(it, provider.storedInfo)
			assert.False(it, provider.storedInfo.GetActive())
			assert.False(it, provider.storedInfo.GetValidated())
			err = revert(provider)
			assert.ErrorIs(it, err, errox.InvariantViolation)
			assert.Nil(it, provider.storedInfo)
			assert.False(it, provider.storedInfo.GetActive())
			assert.False(it, provider.storedInfo.GetValidated())
		})
	}
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
