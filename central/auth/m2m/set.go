package m2m

import (
	"context"
	"testing"

	pkgErrors "github.com/pkg/errors"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	t *tokenExchangerSet

	_ TokenExchangerSet = (*tokenExchangerSet)(nil)
)

// TokenExchangerSet holds token exchangers created from storage.AuthMachineToMachineConfigs.
//
//go:generate mockgen-wrapper
type TokenExchangerSet interface {
	NewTokenExchangerFromConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) (TokenExchanger, error)
	UpsertTokenExchanger(exchanger TokenExchanger, issuer string) error
	RemoveTokenExchanger(issuer string) error
	GetTokenExchanger(issuer string) (TokenExchanger, bool)
}

// TokenExchangerFactory factory for creating a new token exchanger.
type TokenExchangerFactory = func(ctx context.Context, config *storage.AuthMachineToMachineConfig, roleDS roleDataStore.DataStore, issuerFactory tokens.IssuerFactory) (TokenExchanger, error)

// TokenExchangerSetSingleton creates a singleton holding all token exchangers for auth machine to machine configs.
func TokenExchangerSetSingleton(roleDS roleDataStore.DataStore, issuerFactory tokens.IssuerFactory) TokenExchangerSet {
	once.Do(func() {
		t = &tokenExchangerSet{
			tokenExchangers:       map[string]TokenExchanger{},
			roleDS:                roleDS,
			issuerFactory:         issuerFactory,
			tokenExchangerFactory: newTokenExchanger,
		}
	})
	return t
}

// TokenExchangerSetForTesting creates a token set for testing purposes.
func TokenExchangerSetForTesting(_ *testing.T, roleDS roleDataStore.DataStore, issuerFactory tokens.IssuerFactory,
	tokenExchangerFactory TokenExchangerFactory) TokenExchangerSet {
	return &tokenExchangerSet{
		tokenExchangers:       map[string]TokenExchanger{},
		roleDS:                roleDS,
		issuerFactory:         issuerFactory,
		tokenExchangerFactory: tokenExchangerFactory,
	}
}

type tokenExchangerSet struct {
	tokenExchangers       map[string]TokenExchanger
	roleDS                roleDataStore.DataStore
	issuerFactory         tokens.IssuerFactory
	tokenExchangerFactory TokenExchangerFactory
}

// NewTokenExchangerFromConfig creates a TokenExchanger from the given config.
// It will make sure the TokenExchanger is registered as a tokens.Source.
// Note that during creation of the TokenExchanger, an external HTTP request is done to the OIDC issuer
// to retrieve additional metadata for token validation (i.e. JWKS metadata).
func (t *tokenExchangerSet) NewTokenExchangerFromConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) (TokenExchanger, error) {
	exchanger, exists := t.tokenExchangers[config.GetIssuer()]
	if exists {
		// Need to unregister the source temporarily, otherwise we receive an error on creation that the
		// source is already registered.
		if err := t.issuerFactory.UnregisterSource(exchanger.Provider()); err != nil {
			return nil, pkgErrors.Wrapf(err, "unregistering source for config %s", config.GetId())
		}
	}

	tokenExchanger, err := t.tokenExchangerFactory(ctx, config, t.roleDS, t.issuerFactory)
	if err != nil {
		return nil, pkgErrors.Wrapf(err, "creating token exchanger for config %s", config.GetId())
	}

	return tokenExchanger, nil
}

// GetTokenExchanger retrieves a TokenExchanger based on the issuer.
func (t *tokenExchangerSet) GetTokenExchanger(issuer string) (TokenExchanger, bool) {
	tokenExchanger, exists := t.tokenExchangers[issuer]
	return tokenExchanger, exists
}

// UpsertTokenExchanger upserts a token exchanger based off the given config.
// In case a token exchanger already exists for the given config, it will be replaced.
func (t *tokenExchangerSet) UpsertTokenExchanger(exchanger TokenExchanger, issuer string) error {
	t.tokenExchangers[issuer] = exchanger
	return nil
}

// RemoveTokenExchanger removes the token exchanger for the specific configuration ID.
// In case no config with the specific ID exists, nil will be returned.
func (t *tokenExchangerSet) RemoveTokenExchanger(id string) error {
	exchanger, exists := t.tokenExchangers[id]
	if !exists {
		return nil
	}

	// We need to unregister the source with the issuer factory.
	// This will lead to all tokens issued by the previous token exchanger to be rejected by Central.
	if err := t.issuerFactory.UnregisterSource(exchanger.Provider()); err != nil {
		return pkgErrors.Wrapf(err, "unregistering source for config %s", id)
	}

	delete(t.tokenExchangers, id)
	return nil
}
