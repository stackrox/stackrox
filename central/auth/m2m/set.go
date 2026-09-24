package m2m

import (
	"context"
	"fmt"
	"slices"
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

// TokenExchangerSet holds token exchangers created from storage.AuthMachineToMachineConfig objects.
//
//go:generate mockgen-wrapper
type TokenExchangerSet interface {
	UpsertTokenExchanger(issuer string, exchanger TokenExchanger) error
	RemoveTokenExchanger(issuer string) error
	GetTokenExchanger(issuer string) (TokenExchanger, bool)
	HasExchangersConfigured() bool
	// Config management functions
	RemoveConfigForIssuer(ctx context.Context, issuer string, configID string) error
	UpsertConfigForIssuer(ctx context.Context, issuer string, config *storage.AuthMachineToMachineConfig) error
}

// TokenExchangerFactory is a factory for creating a new token exchanger based on multiple m2m configurations.
type TokenExchangerFactory = func(ctx context.Context, configs []*storage.AuthMachineToMachineConfig, roleDS roleDataStore.DataStore, issuerFactory tokens.IssuerFactory) (TokenExchanger, error)

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
func TokenExchangerSetForTesting(
	_ *testing.T,
	roleDS roleDataStore.DataStore,
	issuerFactory tokens.IssuerFactory,
	tokenExchangerFactory TokenExchangerFactory,
) TokenExchangerSet {
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

// UpsertTokenExchanger upserts a token exchanger.
// It will make sure the TokenExchanger is registered as a tokens.Source.
// In case a token exchanger already exists for the given config, it will be replaced.
func (t *tokenExchangerSet) UpsertTokenExchanger(issuer string, exchanger TokenExchanger) error {
	oldExchanger, exists := t.tokenExchangers[issuer]
	if exists {
		// Need to unregister the source temporarily, otherwise we receive an error on creation that the
		// source is already registered.
		if err := t.issuerFactory.UnregisterSource(oldExchanger.Provider()); err != nil {
			return pkgErrors.Wrapf(err, "unregistering source for issuer %s", issuer)
		}
	}
	fmt.Println(exchanger.GetType().String())

	if exchanger.GetType() == storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT {
		if serviceAccountIssuer, _ := GetKubernetesIssuer(); serviceAccountIssuer != "" {
			t.tokenExchangers[serviceAccountIssuer] = exchanger
		}
	}
	t.tokenExchangers[issuer] = exchanger
	fmt.Printf("%p\n", t.issuerFactory)
	fmt.Printf("%p\n", exchanger.Provider())
	if err := t.issuerFactory.RegisterSource(exchanger.Provider()); err != nil {
		return pkgErrors.Wrapf(err, "registering source for issuer %s", issuer)
	}

	return nil
}

// RemoveConfigForIssuer upserts a token exchanger based from the existing configs,
// without the removed config. If the existing configs boil down to the removed one,
// the exchanger for the issuer is removed.
// It will make sure the TokenExchanger is registered as a tokens.Source.
// In case a token exchanger already exists for the given issuer, it will be replaced.
func (t *tokenExchangerSet) RemoveConfigForIssuer(
	ctx context.Context,
	issuer string,
	configID string,
) error {
	oldExchanger, found := t.GetTokenExchanger(issuer)
	if !found {
		return nil
	}
	newExchangerConfigs := slices.DeleteFunc(oldExchanger.Configs(), func(cfg *storage.AuthMachineToMachineConfig) bool {
		return cfg.GetId() == configID
	})
	if len(newExchangerConfigs) == 0 {
		err := t.RemoveTokenExchanger(issuer)
		return err
	}
	updatedTokenExchanger, err := t.tokenExchangerFactory(ctx, newExchangerConfigs, t.roleDS, t.issuerFactory)
	if err != nil {
		return err
	}
	err = t.UpsertTokenExchanger(issuer, updatedTokenExchanger)
	if err != nil {
		return err
	}
	return nil
}

// UpsertConfigForIssuer upserts a token exchanger based on the existing and given configs.
// It will make sure the TokenExchanger is registered as a tokens.Source.
// Note that during creation of the TokenExchanger, an external HTTP request is done to the OIDC issuer
// to retrieve additional metadata for token validation (i.e. JWKS metadata).
// In case a token exchanger already exists for the given config, it will be replaced.
func (t *tokenExchangerSet) UpsertConfigForIssuer(
	ctx context.Context,
	issuer string,
	config *storage.AuthMachineToMachineConfig,
) error {
	oldExchanger, found := t.GetTokenExchanger(issuer)
	var newExchangerConfigs []*storage.AuthMachineToMachineConfig
	if found && oldExchanger != nil {
		newExchangerConfigs = slices.DeleteFunc(oldExchanger.Configs(), func(cfg *storage.AuthMachineToMachineConfig) bool {
			return cfg.GetId() == config.GetId()
		})
	}
	newExchangerConfigs = append(newExchangerConfigs, config)
	updatedExchanger, err := t.tokenExchangerFactory(ctx, newExchangerConfigs, t.roleDS, t.issuerFactory)
	if err != nil {
		return err
	}
	fmt.Println(issuer)
	fmt.Printf("Updated exchanger %p\n", updatedExchanger)
	err = t.UpsertTokenExchanger(issuer, updatedExchanger)
	if err != nil {
		return err
	}
	return nil
}

// GetTokenExchanger retrieves a TokenExchanger based on the issuer.
func (t *tokenExchangerSet) GetTokenExchanger(issuer string) (TokenExchanger, bool) {
	if tokenExchanger, exists := t.tokenExchangers[issuer]; exists {
		return tokenExchanger, true
	}
	return nil, false
}

func (t *tokenExchangerSet) RemoveTokenExchanger(id string) error {
	oldExchanger, exists := t.tokenExchangers[id]

	if !exists {
		return nil
	}

	// We need to unregister the source with the issuer factory.
	// This will lead to all tokens issued by the previous token exchanger to be rejected by Central.
	if err := t.issuerFactory.UnregisterSource(oldExchanger.Provider()); err != nil {
		log.Warnf("Unregistering source for config %s failed: %v", id, err)
		return nil
	}

	delete(t.tokenExchangers, id)
	return nil
}

// HasExchangersConfigured returns true if there is at least one configured
// exchanger.
func (t *tokenExchangerSet) HasExchangersConfigured() bool {
	return len(t.tokenExchangers) > 0
}
