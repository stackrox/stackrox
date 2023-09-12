package filter

import (
	"context"
	"testing"

	postgresStore "github.com/stackrox/rox/central/group/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// FilterFactory is an interface to generate filtered group fetchers
type FilterFactory interface {
	FilteredRetriever() func(ctx context.Context, filter func(*storage.Group) bool) ([]*storage.Group, error)
}

// GetTestPostgresGroupFilterGenerator returns a generator for filtered group
// retrieval connected to a postgres database.
func GetTestPostgresGroupFilterGenerator(_ *testing.T, db postgres.DB) FilterFactory {
	groupStore := postgresStore.New(db)
	return &filterFactoryImpl{
		store: groupStore,
	}
}

type filterFactoryImpl struct {
	store postgresStore.Store
}

func (f *filterFactoryImpl) FilteredRetriever() func(ctx context.Context, filter func(*storage.Group) bool) ([]*storage.Group, error) {
	return func(ctx context.Context, filter func(*storage.Group) bool) ([]*storage.Group, error) {
		return GetFilteredWithStore(ctx, filter, f.store)
	}
}
