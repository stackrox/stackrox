package updater

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	authM2MDataStore "github.com/stackrox/rox/central/auth/datastore"
	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
)

type authMachineToMachineConfigUpdater struct {
	configDataStore authM2MDataStore.DataStore
	healthDataStore declarativeConfigHealth.DataStore
	idExtractor     types.IDExtractor
	nameExtractor   types.NameExtractor
}

var _ ResourceUpdater = (*authMachineToMachineConfigUpdater)(nil)

func newAuthM2MConfigUpdater(configDataStore authM2MDataStore.DataStore, healthDataStore declarativeConfigHealth.DataStore) ResourceUpdater {
	return &authMachineToMachineConfigUpdater{
		configDataStore: configDataStore,
		healthDataStore: healthDataStore,
		idExtractor:     types.UniversalIDExtractor(),
		nameExtractor:   types.UniversalNameExtractor(),
	}
}

func (u *authMachineToMachineConfigUpdater) Upsert(ctx context.Context, m protocompat.Message) error {
	m2mConfig, ok := m.(*storage.AuthMachineToMachineConfig)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to auth machine to machine config updater: %T", m2mConfig)
	}
	_, err := u.configDataStore.UpsertAuthM2MConfig(ctx, m2mConfig)
	return errors.Wrapf(err, "upserting machine to machine config %q for issuer %q",
		m2mConfig.GetId(), m2mConfig.GetIssuer())
}

func (u *authMachineToMachineConfigUpdater) DeleteResources(
	ctx context.Context,
	resourceIDsToSkip ...string,
) ([]string, error) {
	resourcesToSkip := set.NewFrozenStringSet(resourceIDsToSkip...)
	filteredAuthM2MConfigs := make([]*storage.AuthMachineToMachineConfig, 0)
	filterFunction := func(m2mConfig *storage.AuthMachineToMachineConfig) error {
		if !declarativeconfig.IsDeclarativeOrigin(m2mConfig) {
			return nil
		}
		if resourcesToSkip.Contains(m2mConfig.GetId()) {
			return nil
		}
		filteredAuthM2MConfigs = append(filteredAuthM2MConfigs, m2mConfig)
		return nil
	}
	err := u.configDataStore.ForEachAuthM2MConfig(ctx, filterFunction)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving declarative auth machine to machine configurations")
	}

	var deletionErr *multierror.Error
	deletionFailedIDs := make([]string, 0, len(filteredAuthM2MConfigs))
	for _, m2mConfig := range filteredAuthM2MConfigs {
		if err := u.configDataStore.RemoveAuthM2MConfig(ctx, m2mConfig.GetId()); err != nil {
			deletionErr = multierror.Append(deletionErr, err)
			deletionFailedIDs = append(deletionFailedIDs, m2mConfig.GetId())
			if err := u.healthDataStore.UpdateStatusForDeclarativeConfig(ctx, u.idExtractor(m2mConfig), err); err != nil {
				log.Errorf("Failed to update the declarative config health status %q: %v", m2mConfig.GetId(), err)
			}
			if errors.Is(err, errox.ReferencedByAnotherObject) {
				m2mConfig.GetTraits().SetOrigin(storage.Traits_DECLARATIVE_ORPHANED)
				if _, err := u.configDataStore.UpsertAuthM2MConfig(ctx, m2mConfig); err != nil {
					deletionErr = multierror.Append(
						deletionErr,
						errors.Wrapf(err, "setting origin of m2m config %q to orphaned", m2mConfig.GetId()),
					)
				}
			}
		}
	}
	return deletionFailedIDs, deletionErr.ErrorOrNil()
}
