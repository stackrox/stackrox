package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/processbaselineresults/datastore/internal/store"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
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
	elevatedPreSACReadCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.ProcessWhitelist),
		))
	pWResults, exists, err := d.storage.Get(elevatedPreSACReadCtx, deploymentID)
	if err != nil || !exists {
		return nil, err
	}

	if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(pWResults).Allowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, sac.ErrResourceAccessDenied
	}

	return pWResults, nil
}

func (d *datastoreImpl) DeleteBaselineResults(ctx context.Context, deploymentID string) error {
	elevatedPreSACCheckCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.ProcessWhitelist),
		))
	pWResults, exists, err := d.storage.Get(elevatedPreSACCheckCtx, deploymentID)
	if err != nil || !exists {
		return err
	}

	if ok, err := processBaselineSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(pWResults).Allowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.storage.Delete(ctx, deploymentID)
}
