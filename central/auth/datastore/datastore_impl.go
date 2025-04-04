package datastore

import (
	"context"
	"errors"
	"fmt"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/central/auth/m2m"
	"github.com/stackrox/rox/central/auth/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	_ DataStore = (*datastoreImpl)(nil)

	accessSAC                          = sac.ForResource(resources.Access)
	configControllerServiceAccountName = fmt.Sprintf("system:serviceaccount:%s:config-controller", env.Namespace.Setting())
)

type datastoreImpl struct {
	store         store.Store
	set           m2m.TokenExchangerSet
	issuerFetcher m2m.ServiceAccountIssuerFetcher

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

func (d *datastoreImpl) ForEachAuthM2MConfig(ctx context.Context, fn func(obj *storage.AuthMachineToMachineConfig) error) error {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.forEachAuthM2MConfigNoLock(ctx, fn)
}

func (d *datastoreImpl) forEachAuthM2MConfigNoLock(ctx context.Context, fn func(obj *storage.AuthMachineToMachineConfig) error) error {
	return d.store.Walk(ctx, fn)
}

func (d *datastoreImpl) UpsertAuthM2MConfig(ctx context.Context,
	config *storage.AuthMachineToMachineConfig) (*storage.AuthMachineToMachineConfig, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return d.upsertAuthM2MConfigNoLock(ctx, config)
}

func (d *datastoreImpl) upsertAuthM2MConfigNoLock(ctx context.Context,
	config *storage.AuthMachineToMachineConfig) (*storage.AuthMachineToMachineConfig, error) {
	if err := sac.VerifyAuthzOK(accessSAC.WriteAllowed(ctx)); err != nil {
		return nil, err
	}

	// Get the existing stored config, if any.
	storedConfig, exists, err := d.getAuthM2MConfigNoLock(ctx, config.GetId())
	if err != nil {
		return nil, err
	}

	// Get the existing exchanger for the issuer, if any.
	existingExchanger, _ := d.set.GetTokenExchanger(config.GetIssuer())

	// Create a transaction for the DB operation. Since we can potentially fail during in-memory operations (i.e.
	// upserting the token exchanger in the set or removal) we want to make sure we can rollback DB changes.
	ctx, tx, err := d.store.Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Upsert the token exchanger first, ensuring the config is valid and a token exchanger can be successfully
	// created from it.
	if err := d.set.UpsertTokenExchanger(ctx, config); err != nil {
		return nil, d.wrapRollBackSet(ctx, err, storedConfig, config, existingExchanger)
	}

	// Upsert the config to the DB after the token exchanger has been successfully added.
	if err := d.store.Upsert(ctx, config); err != nil {
		return nil, d.wrapRollback(ctx, tx, err, storedConfig, config, existingExchanger)
	}

	// We need to ensure that any previously existing config is removed from the token exchanger set.
	// Since this updated config may have updated the issuer, we need to fetch the existing, stored config from the
	// database and ensure it's removed properly from the set. We do this at the end since we want the new config
	// to successfully exist beforehand.
	if exists && config.GetIssuer() != storedConfig.GetIssuer() {
		if err := d.set.RemoveTokenExchanger(storedConfig.GetIssuer()); err != nil {
			return nil, d.wrapRollback(ctx, tx, err, storedConfig, config, existingExchanger)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, d.wrapRollback(ctx, tx, err, storedConfig, config, existingExchanger)
	}

	return config, nil
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

	kubeSAIssuer, err := d.issuerFetcher.GetServiceAccountIssuer()
	if err != nil {
		return pkgErrors.Wrap(err, "failed to get service account issuer")
	}

	var tokenExchangerErrors []error
	upserted := false
	upsertTokenExchanger := func(config *storage.AuthMachineToMachineConfig) error {
		if err := d.configureConfigControllerAccess(kubeSAIssuer, config); err != nil {
			return pkgErrors.Wrap(err, "failed to configure config controller access")
		}
		if err := d.set.UpsertTokenExchanger(ctx, config); err != nil {
			tokenExchangerErrors = append(tokenExchangerErrors, err)
		}
		upserted = true
		return nil
	}
	if err := d.forEachAuthM2MConfigNoLock(ctx, upsertTokenExchanger); err != nil {
		return pkgErrors.Wrap(err, "Failed to list auth m2m configs")
	}
	// ensure we upserted the default config
	if !upserted {
		if err := d.configureConfigControllerAccess(kubeSAIssuer, nil); err != nil {
			return pkgErrors.Wrap(err, "failed to configure config controller access")
		}
	}

	return errors.Join(tokenExchangerErrors...)
}

// configureConfigControllerAccess ensures the config-controller has access to Central APIs via k8s service account token m2m auth
//
// What this function does in plain english:
//
// * See if any existing m2m configs from the db are for the kube sa issuer
// * If yes, make sure the role mapping for config-controller is present
// * If no, create a new m2m config for kube sa issuer like we do today and save it to the db
//
// This allows customers to add their own role mappings for this config.
// If a customer breaks config-controller auth, they can simply restart Central to get it back to a working state.
func (d *datastoreImpl) configureConfigControllerAccess(kubeSAIssuer string, config *storage.AuthMachineToMachineConfig) error {
	var kubeSAConfig *storage.AuthMachineToMachineConfig

	if config.GetIssuer() == kubeSAIssuer {
		kubeSAConfig = config
	}

	if kubeSAConfig == nil {
		kubeSAConfig = &storage.AuthMachineToMachineConfig{
			Id:                      uuid.NewV4().String(),
			Type:                    storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
			TokenExpirationDuration: "1h",
			Mappings:                []*storage.AuthMachineToMachineConfig_Mapping{},
			Issuer:                  kubeSAIssuer,
		}
	}

	var mappingFound bool
	for _, mapping := range kubeSAConfig.Mappings {
		if mapping.Key == "sub" && mapping.ValueExpression == configControllerServiceAccountName && mapping.Role == "Configuration Controller" {
			mappingFound = true
			break
		}
	}

	if !mappingFound {
		kubeSAConfig.Mappings = append(kubeSAConfig.Mappings, &storage.AuthMachineToMachineConfig_Mapping{
			Key:             "sub",
			ValueExpression: configControllerServiceAccountName,
			Role:            "Configuration Controller",
		})
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resources.Access)))

	// This inits the token exchanger, too
	_, err := d.upsertAuthM2MConfigNoLock(ctx, kubeSAConfig)
	if err != nil {
		return pkgErrors.Wrap(err, "Failed to upsert auth m2m config")
	}

	return nil
}

// wrapRollback wraps the error with potential rollback errors.
// In the case the giving config is not nil, it will attempt to rollback the exchanger in the set in addition to
// rolling back the transaction.
func (d *datastoreImpl) wrapRollback(ctx context.Context, tx *pgPkg.Tx, err error,
	storedConfig, newConfig *storage.AuthMachineToMachineConfig, existingExchanger m2m.TokenExchanger) error {
	err = d.wrapRollBackSet(ctx, err, storedConfig, newConfig, existingExchanger)
	rollbackErr := tx.Rollback(ctx)
	if rollbackErr != nil {
		err = pkgErrors.Wrapf(rollbackErr, "rolling back due to: %v", err)
	}

	return err
}

func (d *datastoreImpl) wrapRollBackSet(ctx context.Context, err error, storedConfig,
	newConfig *storage.AuthMachineToMachineConfig, existingExchanger m2m.TokenExchanger) error {
	exchangerErr := d.set.RemoveTokenExchanger(newConfig.GetIssuer())

	// We have two configs to restore from: either a config has already existed within the DB for the given ID,
	// or a token exchanger exists for the given issuer.
	// We first attempt to restore the exchanger from the stored config. This is the case where e.g. an update to
	// an existing config rendered as invalid.
	// If no stored config is given, i.e. this is was not an update to an existing config, we rollback the changes
	// from the existing exchanger config. This is the case where e.g. a new config was added with the same issuer
	// as an existing config.
	if storedConfig != nil {
		exchangerErr = errors.Join(exchangerErr, d.set.RollbackExchanger(ctx, storedConfig))
	} else if existingExchanger != nil && existingExchanger.Config() != nil {
		exchangerErr = errors.Join(exchangerErr, d.set.RollbackExchanger(ctx, existingExchanger.Config()))
	}

	if exchangerErr != nil {
		err = pkgErrors.Wrapf(exchangerErr, "rolling back due to: %v", err)
	}

	return err
}
