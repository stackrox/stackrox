package updater

import (
	"context"

	"github.com/gogo/protobuf/proto"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

type groupUpdater struct {
	groupDS groupDataStore.DataStore
}

var _ ResourceUpdater = (*groupUpdater)(nil)

func newGroupUpdater(datastore groupDataStore.DataStore) ResourceUpdater {
	return &groupUpdater{
		groupDS: datastore,
	}
}

func (u *groupUpdater) Upsert(ctx context.Context, m proto.Message) error {
	group, ok := m.(*storage.Group)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to group updater: %T", group)
	}
	return u.groupDS.Upsert(ctx, group)
}
