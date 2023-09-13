package filter

import (
	"context"
	"testing"

	postgresStore "github.com/stackrox/rox/central/group/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// RetrieverFactory is an interface to generate filtered group fetchers.
type RetrieverFactory interface {
	NewFilteredRetriever() Retriever
}

// GetTestPostgresGroupFilterGenerator returns a generator for filtered group
// retrieval connected to a postgres database.
func GetTestPostgresGroupFilterGenerator(_ *testing.T, db postgres.DB) RetrieverFactory {
	groupStore := postgresStore.New(db)
	return &factoryImpl{
		store: groupStore,
	}
}

type factoryImpl struct {
	store postgresStore.Store
}

func (f *factoryImpl) NewFilteredRetriever() Retriever {
	return func(ctx context.Context, filter Filter) ([]*storage.Group, error) {
		return GetFilteredWithStore(ctx, filter, f.store)
	}
}
