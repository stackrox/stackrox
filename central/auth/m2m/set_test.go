package m2m

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	authProviderMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/auth/tokens"
	tokensMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTokenExchanger is a minimal TokenExchanger to smoke test the tokenExchangerSet
// without depending on OIDC metadata discovery performed by newTokenExchanger.
type fakeTokenExchanger struct {
	provider authproviders.Provider
}

func (f *fakeTokenExchanger) ExchangeToken(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (f *fakeTokenExchanger) Provider() authproviders.Provider {
	return f.provider
}

func (f *fakeTokenExchanger) Config() *storage.AuthMachineToMachineConfig {
	return nil
}

type testExchangerFactory struct {
	t            testing.TB
	exchanger    TokenExchanger
	factoryError error
	roleStore    roleDataStore.DataStore
}

func (f *testExchangerFactory) SetOutputs(exchanger TokenExchanger, expectedError error) {
	f.exchanger = exchanger
	f.factoryError = expectedError
}

func (f *testExchangerFactory) CreateExchanger(
	_ context.Context,
	_ *storage.AuthMachineToMachineConfig,
	ds roleDataStore.DataStore,
	_ tokens.IssuerFactory,
) (TokenExchanger, error) {
	// The set must forward its own dependencies to the factory.
	assert.Same(f.t, f.roleStore, ds)
	return f.exchanger, f.factoryError
}

func TestUpsertTokenExchanger(t *testing.T) {
	const issuer = "https://issuer.stackrox.example.com"

	newM2MConfig := func() *storage.AuthMachineToMachineConfig {
		return &storage.AuthMachineToMachineConfig{
			Id:                      "test-config-id",
			Type:                    storage.AuthMachineToMachineConfig_GENERIC,
			Issuer:                  issuer,
			TokenExpirationDuration: "1h",
		}
	}

	globalMockCtrl := gomock.NewController(t)
	authProvider := authProviderMocks.NewMockProvider(globalMockCtrl)
	preExistingExchanger := &fakeTokenExchanger{provider: authProvider}

	cases := map[string]struct {
		previousExchanger       TokenExchanger
		seedExisting            bool
		unregisterExistingError error
		factoryError            error
		expectedErrorText       string
		expectExistingRetained  bool
		expectNewExchanger      bool
	}{
		"Exchanger for a new config is added to the set": {
			previousExchanger:  nil,
			expectNewExchanger: true,
		},
		"Exchanger for an existing config is replaced": {
			previousExchanger:  preExistingExchanger,
			expectNewExchanger: true,
		},
		"Failure to unregister the existing exchanger is propagated": {
			previousExchanger:       preExistingExchanger,
			unregisterExistingError: assert.AnError,
			expectedErrorText:       "unregistering source for issuer",
			expectExistingRetained:  true,
		},
		"Failure to create the token exchanger is propagated": {
			previousExchanger: nil,
			factoryError:      assert.AnError,
			expectedErrorText: "creating token exchanger for issuer",
		},
	}

	for name, c := range cases {
		t.Run(name, func(it *testing.T) {
			ctrl := gomock.NewController(t)
			issuerFactory := tokensMocks.NewMockIssuerFactory(ctrl)
			roleDS := roleMocks.NewMockDataStore(ctrl)

			exchangerFactory := &testExchangerFactory{
				t: it,
				roleStore: roleDS,
			}
			set := TokenExchangerSetForTesting(it, roleDS, issuerFactory, exchangerFactory.CreateExchanger)

			if c.previousExchanger != nil {
				exchangerFactory.SetOutputs(c.previousExchanger, nil)
				require.NoError(it, set.UpsertTokenExchanger(context.Background(), newM2MConfig()))

				// Replacing the exchanger first unregisters the source of the existing one.
				issuerFactory.EXPECT().UnregisterSource(c.previousExchanger.Provider()).
					Return(c.unregisterExistingError)
			}

			newExchanger := &fakeTokenExchanger{provider: authProviderMocks.NewMockProvider(ctrl)}
			exchangerFactory.SetOutputs(newExchanger, c.factoryError)

			err := set.UpsertTokenExchanger(context.Background(), newM2MConfig())

			if c.expectedErrorText != "" {
				assert.ErrorContains(it, err, c.expectedErrorText)
				assert.ErrorIs(it, err, assert.AnError)
			} else {
				assert.NoError(it, err)
			}

			retrievedExchanger, ok := set.GetTokenExchanger(issuer)
			if c.expectNewExchanger {
				assert.True(it, ok)
				assert.Same(it, newExchanger, retrievedExchanger)
			} else if c.expectExistingRetained {
				assert.True(it, ok)
				assert.Same(it, c.previousExchanger, retrievedExchanger)
			} else {
				assert.False(it, ok)
				assert.Nil(it, retrievedExchanger)
			}
			assert.Equal(it, c.expectNewExchanger || c.expectExistingRetained, set.HasExchangersConfigured())
		})
	}
}
