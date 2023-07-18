package updater

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/set"
)

type groupUpdater struct {
	groupDS       groupDataStore.DataStore
	healthDS      declarativeConfigHealth.DataStore
	idExtractor   types.IDExtractor
	nameExtractor types.NameExtractor
}

var _ ResourceUpdater = (*groupUpdater)(nil)

func newGroupUpdater(datastore groupDataStore.DataStore, healthDS declarativeConfigHealth.DataStore) ResourceUpdater {
	return &groupUpdater{
		groupDS:       datastore,
		healthDS:      healthDS,
		idExtractor:   types.UniversalIDExtractor(),
		nameExtractor: types.UniversalNameExtractor(),
	}
}

func (u *groupUpdater) Upsert(ctx context.Context, m proto.Message) error {
	group, ok := m.(*storage.Group)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to group updater: %T", group)
	}
	return u.groupDS.Upsert(ctx, group)
}

func (u *groupUpdater) DeleteResources(ctx context.Context, resourceIDsToSkip ...string) ([]string, error) {
	groupsToSkip := set.NewFrozenStringSet(resourceIDsToSkip...)

	groups, err := u.groupDS.GetFiltered(ctx, func(group *storage.Group) bool {
		return declarativeconfig.IsDeclarativeOrigin(group.GetProps()) &&
			!groupsToSkip.Contains(group.GetProps().GetId())
	})
	if err != nil {
		return nil, errors.Wrap(err, "retrieving declarative groups")
	}

	var groupDeletionErr *multierror.Error
	var groupIDs []string
	for _, group := range groups {
		if err := u.groupDS.Remove(ctx, group.GetProps(), true); err != nil {
			groupDeletionErr = multierror.Append(groupDeletionErr, err)
			groupIDs = append(groupIDs, group.GetProps().GetId())
			if err := u.healthDS.UpdateStatusForDeclarativeConfig(ctx, u.idExtractor(group), err); err != nil {
				log.Errorf("Failed to update the declarative config health status %q: %v",
					group.GetProps().GetId(), err)
			}
		}
	}
	return groupIDs, groupDeletionErr.ErrorOrNil()
}
