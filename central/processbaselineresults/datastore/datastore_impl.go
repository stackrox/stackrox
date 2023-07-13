package datastore

import (
	"context"

	"github.com/stackrox/rox/central/processbaselineresults/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	deploymentExtensionSAC = sac.ForResource(resources.DeploymentExtension)
)

type datastoreImpl struct {
	storage store.Store
}

func (d *datastoreImpl) UpsertBaselineResults(ctx context.Context, results *storage.ProcessBaselineResults) error {
	if !deploymentExtensionSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(results).IsAllowed() {
		return sac.ErrResourceAccessDenied
	}

	return d.storage.Upsert(ctx, results)
}

func (d *datastoreImpl) GetBaselineResults(ctx context.Context, deploymentID string) (*storage.ProcessBaselineResults, error) {
	elevatedPreSACReadCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension),
		))
	pWResults, exists, err := d.storage.Get(elevatedPreSACReadCtx, deploymentID)
	if err != nil || !exists {
		return nil, err
	}

	if !deploymentExtensionSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(pWResults).IsAllowed() {
		return nil, sac.ErrResourceAccessDenied
	}

	return pWResults, nil
}

func (d *datastoreImpl) DeleteBaselineResults(ctx context.Context, deploymentID string) error {
	elevatedPreSACCheckCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension),
		))
	pWResults, exists, err := d.storage.Get(elevatedPreSACCheckCtx, deploymentID)
	if err != nil || !exists {
		return err
	}

	if !deploymentExtensionSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(pWResults).IsAllowed() {
		return sac.ErrResourceAccessDenied
	}

	return d.storage.Delete(ctx, deploymentID)
}
