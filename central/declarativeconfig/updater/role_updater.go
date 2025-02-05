package updater

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	declarativeConfigUtils "github.com/stackrox/rox/central/declarativeconfig/utils"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
)

type roleUpdater struct {
	roleDS        roleDataStore.DataStore
	healthDS      declarativeConfigHealth.DataStore
	idExtractor   types.IDExtractor
	nameExtractor types.NameExtractor
}

var _ ResourceUpdater = (*roleUpdater)(nil)

func newRoleUpdater(roleDatastore roleDataStore.DataStore, healthDS declarativeConfigHealth.DataStore) ResourceUpdater {
	return &roleUpdater{
		roleDS:        roleDatastore,
		healthDS:      healthDS,
		idExtractor:   types.UniversalIDExtractor(),
		nameExtractor: types.UniversalNameExtractor(),
	}
}

func (u *roleUpdater) Upsert(ctx context.Context, m protocompat.Message) error {
	role, ok := m.(*storage.Role)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to role updater: %T", role)
	}
	return u.roleDS.UpsertRole(ctx, role)
}

func (u *roleUpdater) DeleteResources(ctx context.Context, resourceIDsToSkip ...string) ([]string, int, error) {
	rolesToSkip := set.NewFrozenStringSet(resourceIDsToSkip...)

	roles, err := u.roleDS.GetRolesFiltered(ctx, func(role *storage.Role) bool {
		return declarativeconfig.IsDeclarativeOrigin(role) &&
			!rolesToSkip.Contains(role.GetName())
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "retrieving declarative roles")
	}

	var roleDeletionErr *multierror.Error
	var roleNames []string
	deletionCount := 0
	for _, role := range roles {
		if err := u.roleDS.RemoveRole(ctx, role.GetName()); err != nil {
			roleDeletionErr = multierror.Append(roleDeletionErr, err)
			roleNames = append(roleNames, role.GetName())
			if err := u.healthDS.UpdateStatusForDeclarativeConfig(ctx,
				declarativeConfigUtils.HealthStatusIDForRole(role.GetName()), err); err != nil {
				log.Errorf("Failed to update the declarative config health status %q: %v", role.GetName(), err)
			}
			if errors.Is(err, errox.ReferencedByAnotherObject) {
				role.Traits.Origin = storage.Traits_DECLARATIVE_ORPHANED
				if err = u.roleDS.UpdateRole(ctx, role); err != nil {
					roleDeletionErr = multierror.Append(roleDeletionErr, errors.Wrap(err, "setting origin to orphaned"))
				}
			}
		} else {
			deletionCount++
		}
	}
	return roleNames, deletionCount, roleDeletionErr.ErrorOrNil()
}
