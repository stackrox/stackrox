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

	noInfoProviderCaseName    = "Provider without storedInfo"
	noOverwriteCaseName       = "Provider with previous option data"
	emptyInfoProviderCaseName = "Provider with empty storedInfo"
)

var (
	baseAuthProviderBackendConfig = map[string]string{
		"baseKey": "baseValue",
	}

	testAuthProviderBackendConfig = map[string]string{
		"testKey": "testValue",
	}

	noIssuer         tokens.Issuer          = nil
	noMapper         permissions.RoleMapper = nil
	noBackend        Backend                = nil
	noBackendFactory BackendFactory         = nil
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

func TestWithBackendFromFactory(t *testing.T) {
	option := WithBackendFromFactory(context.Background(), testAuthProviderBackendFactory)

	baseAuthProviderBackendFactory := &tstAuthProviderBackendFactory{
		ID: baseAuthProviderBackendFactoryID,
	}
	baseAuthProviderBackend := &tstAuthProviderBackend{
		ID: baseAuthProviderBackendID,
	}

	t.Run("Provider with previous backend", func(it *testing.T) {
		provider := &providerImpl{
			backend:        baseAuthProviderBackend,
			backendFactory: baseAuthProviderBackendFactory,
			storedInfo: &storage.AuthProvider{
				Config: baseAuthProviderBackendConfig,
			},
		}
		assert.Equal(it, baseAuthProviderBackend, provider.backend)
		assert.Equal(it, baseAuthProviderBackendFactory, provider.backendFactory)
		assert.Equal(it, baseAuthProviderBackendConfig, extractConfig(provider))
		revert, err := option(provider)
		assert.NoError(it, err)
		assert.Equal(it, testAuthProviderBackend, provider.backend)
		assert.Equal(it, testAuthProviderBackendFactory, provider.backendFactory)
		assert.Equal(it, testAuthProviderBackendConfig, extractConfig(provider))
		err = revert(provider)
		assert.NoError(it, err)
		assert.Equal(it, baseAuthProviderBackend, provider.backend)
		assert.Equal(it, baseAuthProviderBackendFactory, provider.backendFactory)
		assert.Equal(it, baseAuthProviderBackendConfig, extractConfig(provider))
	})

	t.Run("Provider without previous backend", func(it *testing.T) {
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

	extractID := func(provider *providerImpl) interface{} {
		return provider.storedInfo.GetId()
	}
	testNoStoredInfoProvider(t, option, errox.InvariantViolation, extractID)

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

	testNoStoredInfoProvider(t, option, errox.InvariantViolation, extractType)

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

	testNoStoredInfoProvider(t, option, errox.InvariantViolation, extractName)

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
			testNoStoredInfoProvider(it, option, errox.InvariantViolation, extractEnabled)
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
			testNoStoredInfoProvider(it, option, errox.InvariantViolation, extractActive, extractValidated)
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
			testNoStoredInfoProvider(it, option, errox.InvariantViolation, extractVisibility)
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

// region test helpers

func testNoStoredInfoProvider(
	t *testing.T,
	option ProviderOption,
	expectedError error,
	extractors ...func(*providerImpl) interface{},
) {
	t.Run(noInfoProviderCaseName, func(it *testing.T) {
		provider := &providerImpl{}
		assert.Nil(it, provider.storedInfo)
		for _, extractor := range extractors {
			assert.Empty(it, extractor(provider))
		}
		revert, err := option(provider)
		assert.ErrorIs(it, err, expectedError)
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
		fmt.Println("calling option")
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

// region field extractors

func extractID(provider *providerImpl) interface{} {
	return provider.storedInfo.GetId()
}

func extractLoginURL(provider *providerImpl) interface{} {
	return provider.storedInfo.GetLoginUrl()
}

func extractConfig(provider *providerImpl) interface{} {
	return provider.storedInfo.GetConfig()
}

func extractType(provider *providerImpl) interface{} {
	return provider.storedInfo.GetType()
}

func extractName(provider *providerImpl) interface{} {
	return provider.storedInfo.GetName()
}

// endregion field extractors

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

// endregion test helpers
