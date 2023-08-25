package filter

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/group/datastore/internal/store"
	"github.com/stackrox/rox/central/group/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sync"
)

// GetFiltered returns groups from the store filtered using filter function.
func GetFiltered(ctx context.Context, filter func(*storage.Group) bool) ([]*storage.Group, error) {
	return GetFilteredWithStore(ctx, filter, GroupStoreSingleton())
}

// GetFilteredWithStore returns groups from the specified store filtered using filter function.
func GetFilteredWithStore(ctx context.Context, filter func(*storage.Group) bool, store store.Store) ([]*storage.Group, error) {
	var groups []*storage.Group
	walkFn := func() error {
		groups = groups[:0]
		return store.Walk(ctx, func(g *storage.Group) error {
			if filter == nil || filter(g) {
				groups = append(groups, g)
			}
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}
	return groups, nil
}

var (
	groupStore store.Store
	once       sync.Once
)

func initialize() {
	groupStore = postgres.New(globaldb.GetPostgres())
}

// GroupStoreSingleton returns the singleton providing access to the roles store.
func GroupStoreSingleton() store.Store {
	once.Do(initialize)
	return groupStore
}
