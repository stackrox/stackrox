package updater

import (
	"context"

	"github.com/gogo/protobuf/proto"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

type permissionSetUpdater struct {
	roleDS roleDataStore.DataStore
}

var _ ResourceUpdater = (*permissionSetUpdater)(nil)

func newPermissionSetUpdater(datastore roleDataStore.DataStore) ResourceUpdater {
	return &permissionSetUpdater{
		roleDS: datastore,
	}
}

func (u *permissionSetUpdater) Upsert(ctx context.Context, m proto.Message) error {
	permissionSet, ok := m.(*storage.PermissionSet)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to permission set updater: %T", permissionSet)
	}
	return u.roleDS.UpsertPermissionSet(ctx, permissionSet)
}
