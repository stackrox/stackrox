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

	noInfoProviderCaseName = "Provider without storedInfo"
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

func testNoStoredInfoProvider(
	t *testing.T,
	option ProviderOption,
	extractors ...func(*providerImpl) interface{},
) {
	t.Run(noInfoProviderCaseName, func(it *testing.T) {
		provider := &providerImpl{}
		assert.Nil(it, provider.storedInfo)
		for _, extractor := range extractors {
			assert.Empty(it, extractor(provider))
		}
		revert, err := option(provider)
		assert.ErrorIs(it, err, errox.InvariantViolation)
		assert.Nil(it, provider.storedInfo)
		for _, extractor := range extractors {
			assert.Empty(it, extractor(provider))
		}
		err = revert(provider)
		assert.NoError(it, err)
		assert.Nil(it, provider.storedInfo)
		for _, extractor := range extractors {
			assert.Empty(it, extractor(provider))
		}
	})
}

func testProviderOptionApplication(
	option ProviderOption,
	originalStoredInfo *storage.AuthProvider,
	targetValue interface{},
	extractors ...func(*providerImpl) interface{},
) func(*testing.T) {
	return func(t *testing.T) {
		provider := &providerImpl{storedInfo: originalStoredInfo.CloneVT()}
		assert.NotNil(t, provider.storedInfo)
		protoassert.Equal(t, originalStoredInfo, provider.storedInfo)
		revert, err := option(provider)
		assert.NoError(t, err)
		assert.NotNil(t, provider.storedInfo)
		for _, extractor := range extractors {
			assert.Equal(t, targetValue, extractor(provider))
		}
		err = revert(provider)
		assert.NoError(t, err)
		assert.NotNil(t, provider.storedInfo)
		protoassert.Equal(t, originalStoredInfo, provider.storedInfo)
	}
}

func testProviderOptionApplicationBreakingRevert(
	option ProviderOption,
	originalStoredInfo *storage.AuthProvider,
	targetValue interface{},
	extractors ...func(*providerImpl) interface{},
) func(*testing.T) {
	return func(t *testing.T) {
		provider := &providerImpl{storedInfo: originalStoredInfo.CloneVT()}
		assert.NotNil(t, provider.storedInfo)
		protoassert.Equal(t, originalStoredInfo, provider.storedInfo)
		revert, err := option(provider)
		assert.NoError(t, err)
		assert.NotNil(t, provider.storedInfo)
		for _, extractor := range extractors {
			assert.Equal(t, targetValue, extractor(provider))
		}
		provider.storedInfo = nil
		assert.Nil(t, provider.storedInfo)
		for _, extractor := range extractors {
			assert.Empty(t, extractor(provider))
		}
		err = revert(provider)
		assert.ErrorIs(t, err, errox.InvariantViolation)
		assert.Nil(t, provider.storedInfo)
		for _, extractor := range extractors {
			assert.Empty(t, extractor(provider))
		}
	}
}

func TestWithID(t *testing.T) {
	option := WithID(testProviderID)

	extractID := func(provider *providerImpl) interface{} {
		return provider.storedInfo.GetId()
	}
	testNoStoredInfoProvider(t, option, extractID)

	testCases := map[string]*storage.AuthProvider{
		"Provider with storedInfo but no ID": {},
		"Provider with storedInfo and previous ID": {
			Id: authProviderID,
		},
	}

	for name, storedInfo := range testCases {
		t.Run(name, testProviderOptionApplication(option, storedInfo, testProviderID, extractID))
		t.Run(
			name+" - breaking revert",
			testProviderOptionApplicationBreakingRevert(option, storedInfo, testProviderID, extractID),
		)
	}
}

func TestWithType(t *testing.T) {
	option := WithType(testAuthProviderType)

	extractType := func(provider *providerImpl) interface{} {
		return provider.storedInfo.GetType()
	}
	testNoStoredInfoProvider(t, option, extractType)

	testCases := map[string]*storage.AuthProvider{
		"Provider with storedInfo but no type": {},
		"Provider with storedInfo and previous type": {
			Type: baseAuthProviderType,
		},
	}

	for name, storedInfo := range testCases {
		t.Run(name, testProviderOptionApplication(option, storedInfo, testAuthProviderType, extractType))
		t.Run(
			name+" - breaking revert",
			testProviderOptionApplicationBreakingRevert(option, storedInfo, testAuthProviderType, extractType),
		)
	}
}

func TestWithName(t *testing.T) {
	option := WithName(testAuthProviderName)

	extractName := func(provider *providerImpl) interface{} {
		return provider.storedInfo.GetName()
	}
	testNoStoredInfoProvider(t, option, extractName)

	testCases := map[string]*storage.AuthProvider{
		"Provider with storedInfo but no name": {},
		"Provider with storedInfo and previous name": {
			Name: baseAuthProviderName,
		},
	}

	for name, providerStoredInfo := range testCases {
		t.Run(name, testProviderOptionApplication(option, providerStoredInfo, testAuthProviderName, extractName))
		t.Run(
			name+" - breaking revert",
			testProviderOptionApplicationBreakingRevert(option, providerStoredInfo, testAuthProviderName, extractName),
		)
	}
}

func TestWithEnabled(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		option := WithEnabled(enabled)
		extractEnabled := func(provider *providerImpl) interface{} {
			return provider.storedInfo.GetEnabled()
		}
		t.Run(fmt.Sprintf("New Enabled %t", enabled), func(it *testing.T) {
			testNoStoredInfoProvider(it, option, extractEnabled)
			testCases := map[string]*storage.AuthProvider{
				"Provider with storedInfo but enable not set": {},
				"Provider with storedInfo and enabled": {
					Enabled: true,
				},
				"Provider with storedInfo and disabled": {
					Enabled: false,
				},
			}
			for name, providerStoredInfo := range testCases {
				it.Run(name, testProviderOptionApplication(option, providerStoredInfo, enabled, extractEnabled))
				it.Run(
					name+" - breaking revert",
					testProviderOptionApplicationBreakingRevert(option, providerStoredInfo, enabled, extractEnabled),
				)
			}
		})
	}
}

func TestWithActive(t *testing.T) {
	for _, activate := range []bool{true, false} {
		option := WithActive(activate)
		extractActive := func(provider *providerImpl) interface{} {
			return provider.storedInfo.GetActive()
		}
		extractValidated := func(provider *providerImpl) interface{} {
			return provider.storedInfo.GetValidated()
		}
		t.Run(fmt.Sprintf("New Active %t", activate), func(it *testing.T) {
			testNoStoredInfoProvider(it, option, extractActive, extractValidated)
			testCases := map[string]*storage.AuthProvider{
				"Provider with storedInfo, but no active nor validated data": {},
				"Provider with storedInfo, active, but no validated data": {
					Active: true,
				},
				"Provider with storedInfo, inactive, but no validated data": {
					Active: false,
				},
				"Provider with storedInfo, validated, but no active data": {
					Validated: true,
				},
				"Provider with storedInfo, not validated, no active data": {
					Validated: false,
				},
				"Provider with storedInfo, active, validated": {
					Active:    true,
					Validated: true,
				},
				"Provider with storedInfo, active, not validated": {
					Active:    true,
					Validated: false,
				},
				"Provider with storedInfo, inactive, validated": {
					Active:    false,
					Validated: true,
				},
				"Provider with storedInfo, inactive, not validated": {
					Active:    false,
					Validated: false,
				},
			}
			for name, providerStoredInfo := range testCases {
				it.Run(name, testProviderOptionApplication(option, providerStoredInfo, activate, extractActive, extractValidated))
				it.Run(
					name+" - breaking revert",
					testProviderOptionApplicationBreakingRevert(option, providerStoredInfo, activate, extractActive, extractValidated),
				)
			}
		})
	}
}

func TestWithVisibility(t *testing.T) {
	for _, visibility := range []storage.Traits_Visibility{
		storage.Traits_VISIBLE, storage.Traits_HIDDEN,
	} {
		option := WithVisibility(visibility)
		extractVisibility := func(provider *providerImpl) interface{} {
			return provider.storedInfo.GetTraits().GetVisibility()
		}
		t.Run(fmt.Sprintf("New Visibility %s", visibility.String()), func(it *testing.T) {
			testNoStoredInfoProvider(it, option, extractVisibility)
			testCases := map[string]*storage.AuthProvider{
				"Provider with storedInfo, no traits": {},
				"Provider with storedInfo, nil traits": {
					Traits: nil,
				},
				"Provider with storedInfo, traits with no visibility info": {
					Traits: &storage.Traits{},
				},
				"Provider with storedInfo, traits with visible": {
					Traits: &storage.Traits{
						Visibility: storage.Traits_VISIBLE,
					},
				},
				"Provider with storedInfo, traits with hidden": {
					Traits: &storage.Traits{
						Visibility: storage.Traits_HIDDEN,
					},
				},
			}
			for name, providerStoredInfo := range testCases {
				it.Run(name, testProviderOptionApplication(option, providerStoredInfo, visibility, extractVisibility))
				it.Run(
					name+" - breaking revert",
					testProviderOptionApplicationBreakingRevert(option, providerStoredInfo, visibility, extractVisibility),
				)
			}
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
