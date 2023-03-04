package updater

import (
	"context"

	"github.com/gogo/protobuf/proto"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
)

type accessScopeUpdater struct {
	roleDS roleDataStore.DataStore
}

var _ ResourceUpdater = (*accessScopeUpdater)(nil)

func newAccessScopeUpdater(datastore roleDataStore.DataStore) ResourceUpdater {
	return &accessScopeUpdater{
		roleDS: datastore,
	}
}

func (u *accessScopeUpdater) Upsert(ctx context.Context, m proto.Message) error {
	accessScope, ok := m.(*storage.SimpleAccessScope)
	if !ok {
		return errox.InvariantViolation.Newf("wrong type passed to access scope updater: %T", accessScope)
	}
	return u.roleDS.UpsertAccessScope(ctx, accessScope)
}
