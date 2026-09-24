package datastore

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/central/auth/m2m"
	"github.com/stackrox/rox/central/auth/store"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

const configControllerM2MAudience = "central.stackrox.io"

var (
	_ DataStore = (*datastoreImpl)(nil)

	accessSAC                          = sac.ForResource(resources.Access)
	configControllerServiceAccountName = fmt.Sprintf("system:serviceaccount:%s:config-controller", env.Namespace.Setting())
)

type datastoreImpl struct {
	store         store.Store
	set           m2m.TokenExchangerSet
	roleDataStore roleDataStore.DataStore

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

	if err := verifyM2MConfigOrigin(ctx, config); err != nil {
		return nil, err
	}

	if config.GetTraits().GetOrigin() == storage.Traits_DECLARATIVE {
		if err := d.verifyReferencedConfigRoles(ctx, config); err != nil {
			return nil, pkgErrors.Wrap(err, "checking the referenced roles for upsert")
		}
	}

	// Get the existing stored config, if any.
	storedConfig, exists, err := d.getAuthM2MConfigNoLock(ctx, config.GetId())
	if err != nil {
		return nil, err
	}

	// Fetch the existing exchangers for rollback purposes
	var oldExchangerForStoredConfig m2m.TokenExchanger
	oldFoundForStored := false
	if exists {
		oldExchangerForStoredConfig, oldFoundForStored = d.set.GetTokenExchanger(storedConfig.GetIssuer())
	}
	oldExchangerForInputConfig, _ := d.set.GetTokenExchanger(config.GetIssuer())

	// Create a transaction for the DB operation. Since we can potentially fail during in-memory operations (i.e.
	// upserting the token exchanger in the set or removal) we want to make sure we can rollback DB changes.
	ctx, tx, err := d.store.Begin(ctx)
	if err != nil {
		return nil, err
	}

	rollback := func(err error) error {
		err = d.rollbackSetTo(
			err,
			storedConfig.GetIssuer(),
			oldExchangerForStoredConfig,
			config.GetIssuer(),
			oldExchangerForInputConfig,
		)
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil {
			err = pkgErrors.Wrapf(rollbackErr, "rolling back due to: %v", err)
		}
		return err
	}

	// Upsert the token exchanger first, ensuring the config is valid and a token exchanger can be successfully
	// created from it.
	if err = d.set.UpsertConfigForIssuer(ctx, config.GetIssuer(), config); err != nil {
		return nil, rollback(err)
	}

	// Upsert the config to the DB after the token exchanger has been successfully added.
	if err = d.store.Upsert(ctx, config); err != nil {
		return nil, rollback(err)
	}

	// We need to ensure that any previously existing config is removed from the token exchanger set.
	// Since this updated config may have updated the issuer, we need to fetch the existing, stored config from the
	// database and ensure it's removed properly from the set. We do this at the end since we want the new config
	// to successfully exist beforehand.
	if oldFoundForStored && storedConfig.GetIssuer() != config.GetIssuer() {
		if err = d.set.RemoveConfigForIssuer(ctx, storedConfig.GetIssuer(), storedConfig.GetId()); err != nil {
			return nil, rollback(err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, rollback(err)
	}

	return config, nil
}

func verifyM2MConfigOrigin(ctx context.Context, config *storage.AuthMachineToMachineConfig) error {
	if !declarativeconfig.CanModifyResource(ctx, config) {
		return errox.NotAuthorized.CausedByf("machine to machine auth config %q's origin is %s, cannot be modified or deleted with the current permission",
			config.GetIssuer(), config.GetTraits().GetOrigin())
	}
	return nil
}

func (d *datastoreImpl) verifyReferencedConfigRoles(ctx context.Context, config *storage.AuthMachineToMachineConfig) error {
	referencedRoleNames := set.NewSet[string]()
	for _, mapping := range config.GetMappings() {
		referencedRoleNames.Add(mapping.GetRole())
	}
	roleNamesToRetrieve := referencedRoleNames.AsSlice()
	retrievedRoles, missedRoles, err := d.roleDataStore.GetManyRoles(ctx, roleNamesToRetrieve)
	if err != nil {
		return pkgErrors.Wrapf(err, "retrieving roles referenced by machine to machine config %q for issuer %q",
			config.GetId(), config.GetIssuer())
	}
	wrongOriginRoles := make([]string, 0, len(roleNamesToRetrieve))
	for _, role := range retrievedRoles {
		if err := declarativeconfig.VerifyReferencedResourceOrigin(role, config, role.GetName(), config.GetIssuer()); err != nil {
			wrongOriginRoles = append(wrongOriginRoles, role.GetName())
		}
	}
	slices.Sort(missedRoles)
	slices.Sort(wrongOriginRoles)
	if len(missedRoles) > 0 && len(wrongOriginRoles) > 0 {
		return errox.InvalidArgs.CausedByf(
			"imperative roles [%s] and missing roles [%s] can't be referenced by non-imperative "+
				"auth machine to machine configuration %q for issuer %q",
			strings.Join(wrongOriginRoles, ","),
			strings.Join(missedRoles, ","),
			config.GetId(),
			config.GetIssuer(),
		)
	}
	if len(missedRoles) > 0 {
		return errox.InvalidArgs.CausedByf(
			"missing roles [%s] can't be referenced by non-imperative "+
				"auth machine to machine configuration %q for issuer %q",
			strings.Join(missedRoles, ","),
			config.GetId(),
			config.GetIssuer(),
		)
	}
	if len(wrongOriginRoles) > 0 {
		return errox.InvalidArgs.CausedByf(
			"imperative roles [%s] can't be referenced by non-imperative "+
				"auth machine to machine configuration %q for issuer %q",
			strings.Join(wrongOriginRoles, ","),
			config.GetId(),
			config.GetIssuer(),
		)
	}
	return nil
}

func (d *datastoreImpl) GetTokenExchanger(ctx context.Context, issuer string) (m2m.TokenExchanger, bool) {
	if err := sac.VerifyAuthzOK(accessSAC.ReadAllowed(ctx)); err != nil {
		return nil, false
	}
	d.mutex.RLock()
	defer d.mutex.RUnlock()
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
	oldExchanger, _ := d.set.GetTokenExchanger(config.GetIssuer())
	if err = d.set.RemoveConfigForIssuer(ctx, config.GetIssuer(), config.GetId()); err != nil {
		return d.rollbackSetTo(err, config.GetIssuer(), oldExchanger, config.GetIssuer(), oldExchanger)
	}

	return d.store.Delete(ctx, id)
}

func (d *datastoreImpl) InitializeTokenExchangers() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Access)))

	kubeSAIssuer, err := m2m.GetKubernetesIssuer()
	if err != nil {
		return pkgErrors.Wrap(err, "failed to get service account issuer")
	}

	var tokenExchangerErrors []error
	var kubeSAConfig *storage.AuthMachineToMachineConfig
	upsertTokenExchanger := func(config *storage.AuthMachineToMachineConfig) error {
		if config.GetIssuer() == kubeSAIssuer {
			kubeSAConfig = config
			return nil
		}

		if err := d.set.UpsertConfigForIssuer(ctx, config.GetIssuer(), config); err != nil {
			tokenExchangerErrors = append(tokenExchangerErrors, err)
		}
		return nil
	}
	if err := d.forEachAuthM2MConfigNoLock(ctx, upsertTokenExchanger); err != nil {
		return pkgErrors.Wrap(err, "Failed to list auth m2m configs")
	}
	if kubeSAConfig == nil {
		kubeSAConfig = newKubeM2MConfig(kubeSAIssuer)
	}
	if err := d.configureConfigControllerAccess(kubeSAConfig); err != nil {
		return pkgErrors.Wrap(err, "failed to configure config controller access")
	}

	return errors.Join(tokenExchangerErrors...)
}

func newKubeM2MConfig(kubeSAIssuer string) *storage.AuthMachineToMachineConfig {
	return &storage.AuthMachineToMachineConfig{
		Id:                      uuid.NewV4().String(),
		Type:                    storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
		TokenExpirationDuration: "1h",
		Mappings:                []*storage.AuthMachineToMachineConfig_Mapping{},
		Issuer:                  kubeSAIssuer,
		Audience:                configControllerM2MAudience,
	}
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
func (d *datastoreImpl) configureConfigControllerAccess(kubeSAConfig *storage.AuthMachineToMachineConfig) error {
	var mappingFound bool
	for _, mapping := range kubeSAConfig.GetMappings() {
		if mapping.GetKey() == "sub" && mapping.GetValueExpression() == configControllerServiceAccountName && mapping.GetRole() == "Configuration Controller" {
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

	// Above we ensure that the config-controller mapping is always present.
	// But we only set the audience on upgraded configs when the config-controller
	// mapping is the only mapping. If users added their own mappings, their tokens
	// may not carry the custom audience, so enforcing it could break their setup.
	if kubeSAConfig.GetAudience() == "" && len(kubeSAConfig.GetMappings()) == 1 {
		kubeSAConfig.Audience = configControllerM2MAudience
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resources.Access)))

	// This inits the token exchanger, too
	if _, err := d.upsertAuthM2MConfigNoLock(ctx, kubeSAConfig); err != nil {
		return pkgErrors.Wrap(err, "Failed to upsert auth m2m config")
	}

	return nil
}

func (d *datastoreImpl) rollbackSetTo(
	err error,
	storedIssuer string,
	oldStoredTokenExchanger m2m.TokenExchanger,
	inputIssuer string,
	oldInputTokenExchanger m2m.TokenExchanger,
) error {
	var setErr error
	if storedIssuer != inputIssuer {
		// First rollback the exchanger for the input issuer
		if oldInputTokenExchanger == nil {
			if removeErr := d.set.RemoveTokenExchanger(inputIssuer); removeErr != nil {
				setErr = errors.Join(setErr, removeErr)
			}
		} else {
			if upsertErr := d.set.UpsertTokenExchanger(inputIssuer, oldInputTokenExchanger); upsertErr != nil {
				setErr = errors.Join(setErr, upsertErr)
			}
		}
	}
	// Then rollback the exchanger for the stored issuer
	if oldStoredTokenExchanger == nil {
		if removeErr := d.set.RemoveTokenExchanger(storedIssuer); removeErr != nil {
			setErr = errors.Join(setErr, removeErr)
		}
	} else {
		if upsertErr := d.set.UpsertTokenExchanger(storedIssuer, oldStoredTokenExchanger); upsertErr != nil {
			setErr = errors.Join(setErr, upsertErr)
		}
	}
	if setErr != nil {
		err = pkgErrors.Wrapf(setErr, "rolling back due to: %v", err)
	}
	return err
}
