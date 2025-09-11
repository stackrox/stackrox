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
	UpsertTokenExchanger(ctx context.Context, config *storage.AuthMachineToMachineConfig) error
	RemoveTokenExchanger(issuer string) error
	GetTokenExchanger(issuer string) (TokenExchanger, bool)
	RollbackExchanger(ctx context.Context, config *storage.AuthMachineToMachineConfig) error
	HasExchangersConfigured() bool
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

// UpsertTokenExchanger upserts a token exchanger based from the given config.
// It will make sure the TokenExchanger is registered as a tokens.Source.
// Note that during creation of the TokenExchanger, an external HTTP request is done to the OIDC issuer
// to retrieve additional metadata for token validation (i.e. JWKS metadata).
// In case a token exchanger already exists for the given config, it will be replaced.
func (t *tokenExchangerSet) UpsertTokenExchanger(ctx context.Context, config *storage.AuthMachineToMachineConfig) error {
	exchanger, exists := t.tokenExchangers[config.GetIssuer()]
	if exists {
		// Need to unregister the source temporarily, otherwise we receive an error on creation that the
		// source is already registered.
		if err := t.issuerFactory.UnregisterSource(exchanger.Provider()); err != nil {
			return pkgErrors.Wrapf(err, "unregistering source for config %s", config.GetId())
		}
	}

	tokenExchanger, err := t.tokenExchangerFactory(ctx, config, t.roleDS, t.issuerFactory)
	if err != nil {
		return pkgErrors.Wrapf(err, "creating token exchanger for config %s", config.GetId())
	}

	t.tokenExchangers[config.GetIssuer()] = tokenExchanger
	return nil
}

// GetTokenExchanger retrieves a TokenExchanger based on the issuer.
func (t *tokenExchangerSet) GetTokenExchanger(issuer string) (TokenExchanger, bool) {
	// This is the issuer of the tokens, found in the secrets of type
	// kubernetes.io/service-account-token
	const kubernetesServiceAccountSecretIssuer = "kubernetes/serviceaccount"
	if issuer == kubernetesServiceAccountSecretIssuer {
		for _, tokenExchanger := range t.tokenExchangers {
			if tokenExchanger.Config().GetType() == storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT {
				return tokenExchanger, true
			}
		}
	}
	tokenExchanger, exists := t.tokenExchangers[issuer]
	return tokenExchanger, exists
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
		log.Warnf("Unregistering source for config %s failed: %v", id, err)
		return nil
	}

	delete(t.tokenExchangers, id)
	return nil
}

// RollbackExchanger is used to roll back any changes made to an existing exchanger.
// In particular, it will ensure that the source is correctly registered.
func (t *tokenExchangerSet) RollbackExchanger(ctx context.Context, config *storage.AuthMachineToMachineConfig) error {
	// In case the config does not exist anymore, re-create it.
	_, exists := t.tokenExchangers[config.GetIssuer()]
	if !exists {
		return t.UpsertTokenExchanger(ctx, config)
	}

	// In case the config still exists, we remove and re-create it to ensure we have the correct state in our set.
	if err := t.RemoveTokenExchanger(config.GetIssuer()); err != nil {
		return err
	}
	return t.UpsertTokenExchanger(ctx, config)
}

// HasExchangersConfigured returns true if there is at least one configured
// exchanger.
func (t *tokenExchangerSet) HasExchangersConfigured() bool {
	return len(t.tokenExchangers) > 0
}
