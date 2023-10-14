package m2m

import (
	"context"
	"errors"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/central/jwt"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	t *tokenExchangerSet
)

// TokenExchangerSet holds token exchangers created from storage.AuthMachineToMachineConfigs.
//
//go:generate mockgen-wrapper
type TokenExchangerSet interface {
	TokenExchanger
	UpsertTokenExchanger(config *storage.AuthMachineToMachineConfig) error
	RemoveTokenExchanger(id string) error
}

type tokenExchangerSet struct {
	mutex           sync.RWMutex
	tokenExchangers map[string]TokenExchanger
	roleDS          roleDataStore.DataStore
}

// TokenExchangerSetSingleton creates a singleton holding all token exchangers for auth machine to machine configs.
func TokenExchangerSetSingleton(roleDS roleDataStore.DataStore) TokenExchangerSet {
	once.Do(func() {
		t = &tokenExchangerSet{tokenExchangers: map[string]TokenExchanger{}, roleDS: roleDS}
	})
	return t
}

// UpsertTokenExchanger upserts a token exchanger based off the given config.
// In case a token exchanger already exists for the given config, it will be replaced.
func (t *tokenExchangerSet) UpsertTokenExchanger(config *storage.AuthMachineToMachineConfig) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	exchanger, exists := t.tokenExchangers[config.GetId()]
	if exists {
		// Need to unregister the source temporarily, otherwise we receive an error on creation that the
		// source is already registered.
		m2mTokenExchanger := exchanger.(*machineToMachineTokenExchanger)
		if err := jwt.IssuerFactorySingleton().UnregisterSource(m2mTokenExchanger.provider); err != nil {
			return pkgErrors.Wrapf(err, "unregistering source for config %s", config.GetId())
		}
	}

	tokenExchanger, err := newTokenExchanger(config, t.roleDS)
	if err != nil {
		return pkgErrors.Wrapf(err, "creating token exchanger for config %s", config.GetId())
	}

	t.tokenExchangers[config.GetId()] = tokenExchanger
	return nil
}

// RemoveTokenExchanger removes the token exchanger for the specific configuration ID.
// In case no config with the specific ID exists, nil will be returned.
func (t *tokenExchangerSet) RemoveTokenExchanger(id string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	exchanger, exists := t.tokenExchangers[id]
	if !exists {
		return nil
	}

	// We need to unregister the source with the issuer factory.
	// This will lead to all tokens issued by the previous token exchanger to be rejected by Central.
	m2mTokenExchanger := exchanger.(*machineToMachineTokenExchanger)
	if err := jwt.IssuerFactorySingleton().UnregisterSource(m2mTokenExchanger.provider); err != nil {
		return pkgErrors.Wrapf(err, "unregistering source for config %s", id)
	}

	delete(t.tokenExchangers, id)
	return nil
}

// ExchangeToken exchanges the raw ID token using all token exchangers contained within the set.
func (t *tokenExchangerSet) ExchangeToken(ctx context.Context, rawIDToken string) (string, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	var exchangeTokenErrs error
	for _, tokenExchanger := range t.tokenExchangers {
		token, err := tokenExchanger.ExchangeToken(ctx, rawIDToken)
		if err == nil {
			return token, nil
		}
		exchangeTokenErrs = errors.Join(exchangeTokenErrs, err)
	}
	return "", exchangeTokenErrs
}
