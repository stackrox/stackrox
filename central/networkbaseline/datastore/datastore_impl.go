package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkbaseline/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
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

func (ds *dataStoreImpl) GetNetworkBaseline(
	ctx context.Context,
	deploymentID string,
) (*storage.NetworkBaseline, bool, error) {
	baseline, found, err := ds.storage.Get(ctx, deploymentID)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := ds.readAllowed(ctx, baseline); err != nil || !ok {
		return nil, false, err
	}
	return baseline, true, nil
}

type clusterIDNSPair struct {
	clusterID string
	namespace string
}

func (ds *dataStoreImpl) UpsertNetworkBaselines(ctx context.Context, baselines []*storage.NetworkBaseline) error {
	// For simplicity, do nothing and return an error unless the context can write all baselines that are passed in.
	allowedScopes := make(map[clusterIDNSPair]struct{})
	for _, baseline := range baselines {
		pair := clusterIDNSPair{clusterID: baseline.GetClusterId(), namespace: baseline.GetNamespace()}
		if _, allowed := allowedScopes[pair]; allowed {
			continue
		}
		if ok, err := ds.writeAllowed(ctx, baseline); err != nil {
			return err
		} else if !ok {
			return sac.ErrResourceAccessDenied
		}
		allowedScopes[pair] = struct{}{}
	}

	return ds.storage.UpsertMany(ctx, baselines)
}

func (ds *dataStoreImpl) UpdateNetworkBaseline(ctx context.Context, baseline *storage.NetworkBaseline) error {
	if ok, err := ds.writeAllowed(ctx, baseline); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	found, err := ds.validateClusterAndNamespaceAgainstExistingBaseline(ctx, baseline)
	if err != nil {
		return errors.Wrapf(err, "updating network baseline %s", baseline.GetDeploymentId())
	}
	if !found {
		return errors.Errorf("updating a baseline that does not exist: %s", baseline.GetDeploymentId())
	}

	if err := ds.storage.Upsert(ctx, baseline); err != nil {
		return errors.Wrapf(err, "updating network baseline %s into storage", baseline.GetDeploymentId())
	}

	return nil
}

// Validate that the baseline's cluster and namespace matches with what we have if it exists
//   - returns true if baseline already exists
//   - returns error if existing baseline does not match with provided baseline
func (ds *dataStoreImpl) validateClusterAndNamespaceAgainstExistingBaseline(
	ctx context.Context,
	baseline *storage.NetworkBaseline,
) (bool, error) {
	existingBaseline, found, err := ds.storage.Get(ctx, baseline.GetDeploymentId())
	if err != nil || !found {
		return false, err
	}
	if existingBaseline.GetClusterId() != baseline.GetClusterId() ||
		existingBaseline.GetNamespace() != baseline.GetNamespace() {
		return true, errors.Errorf(
			"cluster ID %s and namespace %s do not match with existing network baseline",
			baseline.ClusterId,
			baseline.Namespace)
	}
	return true, nil
}

func (ds *dataStoreImpl) DeleteNetworkBaseline(ctx context.Context, deploymentID string) error {
	return ds.DeleteNetworkBaselines(ctx, []string{deploymentID})
}

func (ds *dataStoreImpl) DeleteNetworkBaselines(ctx context.Context, deploymentIDs []string) error {
	// First check permission
	for _, id := range deploymentIDs {
		baseline, found, err := ds.storage.Get(ctx, id)
		if err != nil {
			return err
		} else if !found {
			continue
		}
		if ok, err := ds.writeAllowed(ctx, baseline); err != nil {
			return err
		} else if !ok {
			return sac.ErrResourceAccessDenied
		}
	}

	if err := ds.storage.DeleteMany(ctx, deploymentIDs); err != nil {
		return errors.Wrapf(err, "deleting network baselines %q from storage", deploymentIDs)
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

func (ds *dataStoreImpl) Walk(ctx context.Context, f func(baseline *storage.NetworkBaseline) error) error {
	if ok, err := networkBaselineSAC.ReadAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return nil
	}

	return ds.storage.Walk(ctx, f)

}
