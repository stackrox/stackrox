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
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	// issuerSeparator is used to encode the issuer field for database storage.
	// The stored issuer is the concatenation of the traits origin and the issuer value.
	issuerSeparator = "|"
)

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

// encodeIssuer encodes the issuer field by concatenating the traits origin and the issuer value.
// This allows different origins to have configurations with the same issuer URL.
// Format: "ORIGIN|issuer_url" (e.g., "DECLARATIVE|https://example.com")
func encodeIssuer(config *storage.AuthMachineToMachineConfig) string {
	origin := config.GetTraits().GetOrigin().String()
	issuer := config.GetIssuer()
	return fmt.Sprintf("%s%s%s", origin, issuerSeparator, issuer)
}

// decodeIssuer extracts the raw issuer from an encoded issuer string.
// The encoded format is "ORIGIN|issuer", this returns just the "issuer" part.
func decodeIssuer(encodedIssuer string) string {
	_, issuer, found := strings.Cut(encodedIssuer, issuerSeparator)
	if found {
		return issuer
	}
	// If not encoded, return as-is (for backwards compatibility)
	return encodedIssuer
}

// withEncodedIssuer creates a copy of the config with the issuer field encoded.
// This copy is used for database operations to ensure the unique constraint works per origin.
func withEncodedIssuer(config *storage.AuthMachineToMachineConfig) *storage.AuthMachineToMachineConfig {
	// Clone the config to avoid modifying the original
	encoded := config.CloneVT()
	// Set the encoded issuer
	encoded.Issuer = encodeIssuer(config)
	return encoded
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

	// First, pull the stored configuration being updated, if it exists.
	storedConfig, exists, err := d.getAuthM2MConfigNoLock(ctx, config.GetId())
	if err != nil {
		return nil, err
	}

	// Then, get the associated existing exchanger for the stored configuration's issuer.
	var existingExchanger m2m.TokenExchanger
	var storedConfigs []*storage.AuthMachineToMachineConfig
	if exists && storedConfig != nil {
		// Decode the issuer since the stored config has an encoded issuer but the
		// token exchanger set uses raw issuers as keys.
		rawIssuer := decodeIssuer(storedConfig.GetIssuer())
		existingExchanger, _ = d.set.GetTokenExchanger(rawIssuer)

		// Finally, get the configurations referenced by the exchanger.
		if existingExchanger != nil {
			exchangerConfigs := existingExchanger.Configs()
			storedConfigs = make([]*storage.AuthMachineToMachineConfig, 0, len(exchangerConfigs))
			for _, exchangerConfig := range exchangerConfigs {
				sc, scExists, scErr := d.getAuthM2MConfigNoLock(ctx, exchangerConfig.GetId())
				if scErr != nil {
					return nil, scErr
				}
				if scExists && sc != nil {
					storedConfigs = append(storedConfigs, sc)
				}
			}
		}
	}

	// Determine the issuer for rollback operations (use stored config's issuer if available, otherwise new config's).
	issuer := config.GetIssuer()
	if storedConfig != nil && storedConfig.GetIssuer() != "" {
		issuer = storedConfig.GetIssuer()
	}

	hasDiscrepantIssuer := exists && storedConfig != nil && storedConfig.GetIssuer() != config.GetIssuer()

	// Create a transaction for the DB operation. Since we can potentially fail during in-memory operations (i.e.
	// upserting the token exchanger in the set or removal) we want to make sure we can rollback DB changes.
	ctx, tx, err := d.store.Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Upsert the token exchanger first, ensuring the config is valid and a token exchanger can be successfully
	// created from it.
	if err := d.set.UpsertTokenExchanger(ctx, config); err != nil {
		return nil, d.wrapRollback(ctx, tx, err, issuer, storedConfigs, config, existingExchanger)
	}

	// Upsert the config to the DB after the token exchanger has been successfully added.
	// We use withEncodedIssuer to ensure the database issuer column contains "ORIGIN|issuer"
	// for unique constraint enforcement per origin.
	if err := d.store.Upsert(ctx, withEncodedIssuer(config)); err != nil {
		return nil, d.wrapRollback(ctx, tx, err, issuer, storedConfigs, config, existingExchanger)
	}

	// We need to ensure that any previously existing config is removed from the token exchanger set.
	// Since this updated config may have updated the issuer, we need to remove the stored config from the
	// old exchanger. We do this at the end since we want the new config to successfully exist beforehand.
	if hasDiscrepantIssuer && storedConfig != nil {
		if err := d.set.RemoveTokenExchanger(ctx, storedConfig); err != nil {
			return nil, d.wrapRollback(ctx, tx, err, issuer, storedConfigs, config, existingExchanger)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, d.wrapRollback(ctx, tx, err, issuer, storedConfigs, config, existingExchanger)
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
	if err := d.set.RemoveTokenExchanger(ctx, config); err != nil {
		return err
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

		if err := d.set.UpsertTokenExchanger(ctx, config); err != nil {
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
		Traits:                  &storage.Traits{Origin: storage.Traits_DEFAULT},
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

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resources.Access)))

	// This inits the token exchanger, too
	if _, err := d.upsertAuthM2MConfigNoLock(ctx, kubeSAConfig); err != nil {
		return pkgErrors.Wrap(err, "Failed to upsert auth m2m config")
	}

	return nil
}

// wrapRollback wraps the error with potential rollback errors.
// In the case the giving config is not nil, it will attempt to rollback the exchanger in the set in addition to
// rolling back the transaction.
func (d *datastoreImpl) wrapRollback(
	ctx context.Context,
	tx *pgPkg.Tx,
	err error,
	issuer string,
	storedConfigs []*storage.AuthMachineToMachineConfig,
	newConfig *storage.AuthMachineToMachineConfig,
	existingExchanger m2m.TokenExchanger,
) error {
	err = d.wrapRollBackSet(ctx, err, issuer, storedConfigs, newConfig, existingExchanger)
	rollbackErr := tx.Rollback(ctx)
	if rollbackErr != nil {
		err = pkgErrors.Wrapf(rollbackErr, "rolling back due to: %v", err)
	}

	return err
}

func (d *datastoreImpl) wrapRollBackSet(
	ctx context.Context,
	err error,
	issuer string,
	storedConfigs []*storage.AuthMachineToMachineConfig,
	newConfig *storage.AuthMachineToMachineConfig,
	existingExchanger m2m.TokenExchanger,
) error {
	exchangerErr := d.set.RemoveTokenExchanger(ctx, newConfig)

	// We have two configs to restore from: either a config has already existed within the DB for the given ID,
	// or a token exchanger exists for the given issuer.
	// We first attempt to restore the exchanger from the stored config. This is the case where e.g. an update to
	// an existing config rendered as invalid.
	// If no stored config is given, i.e. this was not an update to an existing config, we rollback the changes
	// from the existing exchanger config. This is the case where e.g. a new config was added with the same issuer
	// as an existing config.
	if len(storedConfigs) > 0 {
		exchangerErr = errors.Join(exchangerErr, d.set.RollbackExchanger(ctx, issuer, storedConfigs))
	} else if existingExchanger != nil && len(existingExchanger.Configs()) > 0 {
		exchangerErr = errors.Join(exchangerErr, d.set.RollbackExchanger(ctx, issuer, existingExchanger.Configs()))
	}

	if exchangerErr != nil {
		err = pkgErrors.Wrapf(exchangerErr, "rolling back due to: %v", err)
	}

	return err
}
