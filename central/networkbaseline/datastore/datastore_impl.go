package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkbaseline/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	networkBaselineSAC = sac.ForResource(resources.NetworkBaseline)
)

type dataStoreImpl struct {
	storage store.Store
}

// newNetworkBaselineDataStore returns a new instance of EntityDataStore using the input storage underneath.
func newNetworkBaselineDataStore(storage store.Store) DataStore {
	ds := &dataStoreImpl{
		storage: storage,
	}
	return ds
}

func (ds *dataStoreImpl) Exists(ctx context.Context, deploymentID string) (bool, error) {
	_, ok, err := ds.GetNetworkBaseline(ctx, deploymentID)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (ds *dataStoreImpl) GetNetworkBaseline(
	ctx context.Context,
	deploymentID string,
) (*storage.NetworkBaseline, bool, error) {
	baseline, found, err := ds.storage.Get(deploymentID)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := ds.readAllowed(ctx, baseline); err != nil || !ok {
		return nil, false, err
	}
	return baseline, true, nil
}

func (ds *dataStoreImpl) UpsertNetworkBaseline(ctx context.Context, baseline *storage.NetworkBaseline) error {
	if ok, err := ds.writeAllowed(ctx, baseline); err != nil {
		return err
	} else if !ok {
		return sac.ErrPermissionDenied
	}

	// Validate that the baseline's cluster and namespace matches with what we have if it exists
	existingBaseline, found, err := ds.storage.Get(baseline.GetDeploymentId())
	if err != nil {
		return err
	}
	if found &&
		(existingBaseline.GetClusterId() != baseline.GetClusterId() ||
			existingBaseline.GetNamespace() != baseline.GetNamespace()) {
		return errors.Errorf(
			"upsert: cluster ID %s and namespace %s do not match with existing network baseline",
			baseline.ClusterId,
			baseline.Namespace)
	}

	if err := ds.storage.Upsert(baseline); err != nil {
		return errors.Wrapf(err, "upserting network baseline %s into storage", baseline.GetDeploymentId())
	}

	return nil
}

func (ds *dataStoreImpl) DeleteNetworkBaseline(ctx context.Context, deploymentID string) error {
	baseline, found, err := ds.storage.Get(deploymentID)
	if err != nil || !found {
		return err
	}

	if ok, err := ds.writeAllowed(ctx, baseline); err != nil {
		return err
	} else if !ok {
		return sac.ErrPermissionDenied
	}

	if err := ds.storage.Delete(deploymentID); err != nil {
		return errors.Wrapf(err, "deleting network baseline %s from storage", deploymentID)
	}

	return nil
}

func (ds *dataStoreImpl) readAllowed(ctx context.Context, baseline *storage.NetworkBaseline) (bool, error) {
	return ds.allowed(ctx, storage.Access_READ_ACCESS, baseline)
}

func (ds *dataStoreImpl) writeAllowed(ctx context.Context, baseline *storage.NetworkBaseline) (bool, error) {
	return ds.allowed(ctx, storage.Access_READ_WRITE_ACCESS, baseline)
}

func (ds *dataStoreImpl) allowed(
	ctx context.Context,
	access storage.Access,
	baseline *storage.NetworkBaseline,
) (bool, error) {
	return networkBaselineSAC.ScopeChecker(ctx, access).ForNamespaceScopedObject(baseline).Allowed(ctx)
}
