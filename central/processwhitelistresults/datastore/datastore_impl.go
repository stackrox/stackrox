package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/processwhitelistresults/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	processWhitelistSAC = sac.ForResource(resources.ProcessWhitelist)
)

type datastoreImpl struct {
	storage store.Store
}

func (d *datastoreImpl) UpsertWhitelistResults(ctx context.Context, results *storage.ProcessWhitelistResults) error {
	if ok, err := processWhitelistSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(results).Allowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return d.storage.UpsertWhitelistResults(results)
}

func (d *datastoreImpl) GetWhitelistResults(ctx context.Context, deploymentID string) (*storage.ProcessWhitelistResults, error) {
	pWResults, err := d.storage.GetWhitelistResults(deploymentID)
	if err != nil {
		return nil, err
	}

	if ok, err := processWhitelistSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(pWResults).Allowed(ctx); err != nil || !ok {
		return nil, err
	}

	return pWResults, nil
}
