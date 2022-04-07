package datastore

import (
	"context"

	"github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	processBaselineSAC = sac.ForResource(resources.ProcessWhitelist)
)

type datastoreImpl struct {
	storage store.Store
}

func (d *datastoreImpl) UpsertBaselineResults(ctx context.Context, results *storage.ProcessBaselineResults) error {
	if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(results).Allowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.storage.Upsert(ctx, results)
}

func (d *datastoreImpl) GetBaselineResults(ctx context.Context, deploymentID string) (*storage.ProcessBaselineResults, error) {
	pWResults, exists, err := d.storage.Get(ctx, deploymentID)
	if err != nil || !exists {
		return nil, err
	}

	if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(pWResults).Allowed(ctx); err != nil || !ok {
		return nil, err
	}

	return pWResults, nil
}

func (d *datastoreImpl) DeleteBaselineResults(ctx context.Context, deploymentID string) error {
	pWResults, exists, err := d.storage.Get(ctx, deploymentID)
	if err != nil || !exists {
		return err
	}

	if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(pWResults).Allowed(ctx); err != nil || !ok {
		return err
	}

	return d.storage.Delete(ctx, deploymentID)
}
