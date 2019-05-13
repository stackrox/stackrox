package datastore

import (
	"context"

	"github.com/stackrox/rox/central/processwhitelistresults/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

type datastoreImpl struct {
	storage store.Store
}

func (d *datastoreImpl) UpsertWhitelistResults(ctx context.Context, results *storage.ProcessWhitelistResults) error {
	return d.storage.UpsertWhitelistResults(results)
}

func (d *datastoreImpl) GetWhitelistResults(ctx context.Context, deploymentID string) (*storage.ProcessWhitelistResults, error) {
	return d.storage.GetWhitelistResults(deploymentID)
}

func (d *datastoreImpl) DeleteWhitelistResults(ctx context.Context, deploymentID string) error {
	return d.storage.DeleteWhitelistResults(deploymentID)
}
