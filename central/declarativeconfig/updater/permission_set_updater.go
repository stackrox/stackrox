package updater

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/set"
)

type permissionSetUpdater struct {
	roleDS        roleDataStore.DataStore
	healthDS      declarativeConfigHealth.DataStore
	idExtractor   types.IDExtractor
	nameExtractor types.NameExtractor
}

var _ ResourceUpdater = (*permissionSetUpdater)(nil)

func newPermissionSetUpdater(datastore roleDataStore.DataStore, healthDS declarativeConfigHealth.DataStore) ResourceUpdater {
	return &permissionSetUpdater{
		roleDS:        datastore,
		healthDS:      healthDS,
		idExtractor:   types.UniversalIDExtractor(),
		nameExtractor: types.UniversalNameExtractor(),
	}
}

func (u *permissionSetUpdater) Upsert(ctx context.Context, m proto.Message) error {
	permissionSet, ok := m.(*storage.PermissionSet)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to permission set updater: %T", permissionSet)
	}
	return u.roleDS.UpsertPermissionSet(ctx, permissionSet)
}

func (u *permissionSetUpdater) DeleteResources(ctx context.Context, resourceIDsToSkip ...string) ([]string, error) {
	permissionSetsToSkip := set.NewFrozenStringSet(resourceIDsToSkip...)

	permissionSets, err := u.roleDS.GetPermissionSetsFiltered(ctx, func(permissionSet *storage.PermissionSet) bool {
		return declarativeconfig.IsDeclarativeOrigin(permissionSet) &&
			!permissionSetsToSkip.Contains(permissionSet.GetId())
	})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving declarative permission sets")
	}

	var permissionSetDeletionErr *multierror.Error
	var permissionSetIDs []string
	for _, permissionSet := range permissionSets {
		if err := u.roleDS.RemovePermissionSet(ctx, permissionSet.GetId()); err != nil {
			permissionSetDeletionErr = multierror.Append(permissionSetDeletionErr, err)
			permissionSetIDs = append(permissionSetIDs, permissionSet.GetId())
			if err := u.healthDS.UpdateStatusForDeclarativeConfig(ctx, u.idExtractor(permissionSet), err); err != nil {
				log.Errorf("Failed to update the declarative config health status %q: %v", permissionSet.GetId(), err)
			}
			if errors.Is(err, errox.ReferencedByAnotherObject) {
				permissionSet.Traits.Origin = storage.Traits_DECLARATIVE_ORPHANED
				if err = u.roleDS.UpsertPermissionSet(ctx, permissionSet); err != nil {
					permissionSetDeletionErr = multierror.Append(permissionSetDeletionErr, errors.Wrap(err, "setting origin to orphaned"))
				}
			}
		}
	}
	return permissionSetIDs, permissionSetDeletionErr.ErrorOrNil()
}
