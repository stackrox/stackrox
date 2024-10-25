package postgres

import (
	"context"

	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store"
	"github.com/stackrox/rox/pkg/postgres"
)

// NewFullStore augments the generated store with RemoveOrphanedEntities function.
func NewFullStore(db postgres.DB) store.EntityStore {
	return &fullStoreImpl{
		Store: New(db),
	}
}

// FullStoreWrap augments the wrapped store with ListDeployment functions.
func FullStoreWrap(wrapped Store) Store {
	return &fullStoreImpl{
		Store: wrapped,
	}
}

type fullStoreImpl struct {
	Store
}

// RemoveOrphanedEntities prunes 'discovered' external entities that are not referenced by any flow.
func (f *fullStoreImpl) RemoveOrphanedEntities(ctx context.Context) error {
	// TODO
	return nil
}
