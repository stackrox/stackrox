package updater

import (
	"context"

	"github.com/gogo/protobuf/proto"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

type roleUpdater struct {
	roleDS roleDataStore.DataStore
}

var _ ResourceUpdater = (*roleUpdater)(nil)

func newRoleUpdater(datastore roleDataStore.DataStore) ResourceUpdater {
	return &roleUpdater{
		roleDS: datastore,
	}
}

func (u *roleUpdater) Upsert(ctx context.Context, m proto.Message) error {
	role, ok := m.(*storage.Role)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to role updater: %T", role)
	}
	return u.roleDS.UpsertRole(ctx, role)
}
