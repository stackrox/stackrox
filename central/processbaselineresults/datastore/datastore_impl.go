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
	return d.storage.Upsert(ctx, results)
}

func (d *datastoreImpl) GetBaselineResults(ctx context.Context, deploymentID string) (*storage.ProcessBaselineResults, error) {
	// The access control behaviour in this function is not standard. It also does not respect
	// some base information security principles (behave the same when the requester
	// should not be allowed to access the information and if the target information
	// does not exist).
	// Backward behaviour consistency for the /v1/deploymentswithprocessinfo endpoint would require
	// this function to stay as is.
	elevatedPreSACReadCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
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
	return d.storage.Delete(ctx, deploymentID)
}
