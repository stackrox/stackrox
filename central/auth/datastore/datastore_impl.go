package datastore

import (
	"context"
	"errors"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/central/auth/m2m"
	"github.com/stackrox/rox/central/auth/store"
	"github.com/stackrox/rox/generated/storage"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	_ DataStore = (*datastoreImpl)(nil)

	accessSAC = sac.ForResource(resources.Access)
)

type datastoreImpl struct {
	store store.Store
	set   m2m.TokenExchangerSet

	mutex sync.RWMutex
}

func (d *datastoreImpl) GetAuthM2MConfig(ctx context.Context, id string) (*storage.AuthMachineToMachineConfig, bool, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.getAuthM2MConfigNoLock(ctx, id)
}

func (d *datastoreImpl) getAuthM2MConfigNoLock(ctx context.Context, id string) (*storage.AuthMachineToMachineConfig, bool, error) {
	return d.store.Get(ctx, id)
}

func (d *datastoreImpl) ListAuthM2MConfigs(ctx context.Context) ([]*storage.AuthMachineToMachineConfig, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.listAuthM2MConfigsNoLock(ctx)
}

func (d *datastoreImpl) listAuthM2MConfigsNoLock(ctx context.Context) ([]*storage.AuthMachineToMachineConfig, error) {
	return d.store.GetAll(ctx)
}

func (d *datastoreImpl) AddAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) (*storage.AuthMachineToMachineConfig, error) {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return nil, err
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	existingConfig, _, err := d.getAuthM2MConfigNoLock(ctx, config.GetId())
	if err != nil {
		return nil, err
	}

	if err := d.set.UpsertTokenExchanger(ctx, config); err != nil {
		return nil, d.wrapRollBackSet(ctx, err, existingConfig, config)
	}

	if err := d.store.Upsert(ctx, config); err != nil {
		return nil, d.wrapRollBackSet(ctx, err, existingConfig, config)
	}

	return config, nil
}

func (d *datastoreImpl) UpdateAuthM2MConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) error {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	existingConfig, exists, err := d.getAuthM2MConfigNoLock(ctx, config.GetId())
	if err != nil {
		return err
	}

	ctx, tx, err := d.store.Begin(ctx)
	if err != nil {
		return err
	}

	if err := d.set.UpsertTokenExchanger(ctx, config); err != nil {
		return d.wrapRollBackSet(ctx, err, existingConfig, config)
	}

	if err := d.store.Upsert(ctx, config); err != nil {
		return d.wrapRollback(ctx, tx, err, existingConfig, config)
	}

	// We need to ensure that any previously existing config is removed from the token exchanger set.
	// Since this updated config may have updated the issuer, we need to fetch the existing, stored config from the
	// database and ensure it's removed properly from the set. We do this at the end since we want the new config
	// to successfully exist beforehand.
	if exists && config.GetIssuer() != existingConfig.GetIssuer() {
		if err := d.set.RemoveTokenExchanger(existingConfig.GetIssuer()); err != nil {
			return d.wrapRollback(ctx, tx, err, existingConfig, config)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return d.wrapRollback(ctx, tx, err, existingConfig, config)
	}

	return nil
}

func (d *datastoreImpl) GetTokenExchanger(ctx context.Context, issuer string) (m2m.TokenExchanger, bool) {
	if err := sac.VerifyAuthzOK(accessSAC.ReadAllowed(ctx)); err != nil {
		return nil, false
	}
	return d.set.GetTokenExchanger(issuer)
}

func (d *datastoreImpl) RemoveAuthM2MConfig(ctx context.Context, id string) error {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	config, exists, err := d.getAuthM2MConfigNoLock(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if err := d.set.RemoveTokenExchanger(config.GetIssuer()); err != nil {
		return err
	}

	return d.store.Delete(ctx, id)
}

func (d *datastoreImpl) InitializeTokenExchangers() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Access)))

	configs, err := d.listAuthM2MConfigsNoLock(ctx)
	if err != nil {
		return err
	}

	var tokenExchangerErrors error
	for _, config := range configs {
		if err := d.set.UpsertTokenExchanger(ctx, config); err != nil {
			tokenExchangerErrors = errors.Join(tokenExchangerErrors, err)
			continue
		}
	}
	if tokenExchangerErrors != nil {
		return tokenExchangerErrors
	}
	return tokenExchangerErrors
}

// wrapRollback wraps the error with potential rollback errors.
// In the case the giving config is not nil, it will attempt to rollback the exchanger in the set in addition to
// rolling back the transaction.
func (d *datastoreImpl) wrapRollback(ctx context.Context, tx *pgPkg.Tx, err error,
	existingConfig *storage.AuthMachineToMachineConfig, newConfig *storage.AuthMachineToMachineConfig) error {
	err = d.wrapRollBackSet(ctx, err, existingConfig, newConfig)
	rollbackErr := tx.Rollback(ctx)
	if rollbackErr != nil {
		err = pkgErrors.Wrapf(rollbackErr, "rolling back due to: %v", err)
	}

	return err
}

func (d *datastoreImpl) wrapRollBackSet(ctx context.Context, err error, existingConfig,
	newConfig *storage.AuthMachineToMachineConfig) error {
	exchangerErr := d.set.RemoveTokenExchanger(newConfig.GetIssuer())

	exchangerErr = errors.Join(exchangerErr, d.set.RollbackExchanger(ctx, existingConfig))

	if exchangerErr != nil {
		err = pkgErrors.Wrapf(exchangerErr, "rolling back due to: %v", err)
	}

	return err
}
