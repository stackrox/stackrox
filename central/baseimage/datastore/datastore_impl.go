package datastore

import (
	"context"

	"github.com/stackrox/rox/central/baseimage/store/repository/postgres"
	"github.com/stackrox/rox/generated/storage"
)

type datastoreImpl struct {
	store postgres.Store
}

// New creates a new DataStore instance backed by PostgreSQL.
func New(store postgres.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}

func (ds *datastoreImpl) ListRepositories(ctx context.Context) ([]*storage.BaseImageRepository, error) {
	var repos []*storage.BaseImageRepository

	err := ds.store.Walk(ctx, func(repo *storage.BaseImageRepository) error {
		repos = append(repos, repo)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return repos, nil
}
