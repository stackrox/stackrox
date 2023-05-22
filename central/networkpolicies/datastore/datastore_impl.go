package store

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	netpolSAC = sac.ForResource(resources.NetworkPolicy)
)

type datastoreImpl struct {
	storage store.Store

	undoStorageLock       sync.Mutex
	undoStorage           undostore.UndoStore
	undoDeploymentStorage undodeploymentstore.UndoDeploymentStore
}

func (ds *datastoreImpl) GetNetworkPolicy(ctx context.Context, id string) (*storage.NetworkPolicy, bool, error) {
	np, found, err := ds.getNetworkPolicy(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	if !netpolSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(np).IsAllowed() {
		return nil, false, nil
	}

	return np, true, nil
}

func (ds *datastoreImpl) doForMatching(ctx context.Context, clusterID, namespace string, fn func(np *storage.NetworkPolicy)) error {
	// Postgres retry in caller.
	return ds.storage.Walk(ctx, func(np *storage.NetworkPolicy) error {
		if clusterID != "" && np.GetClusterId() != clusterID {
			return nil
		}
		if namespace != "" && np.GetNamespace() != namespace {
			return nil
		}
		fn(np)
		return nil
	})
}

func (ds *datastoreImpl) GetNetworkPolicies(ctx context.Context, clusterID, namespace string) ([]*storage.NetworkPolicy, error) {
	var netPols []*storage.NetworkPolicy
	err := pgutils.RetryIfPostgres(
		func() error {
			netPols = netPols[:0]
			return ds.doForMatching(ctx, clusterID, namespace, func(np *storage.NetworkPolicy) {
				netPols = append(netPols, np)
			})
		},
	)
	if err != nil {
		return nil, err
	}
	if namespace == "" {
		return filterResults(ctx, netpolSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS), netPols)
	}

	scopeKeys := []sac.ScopeKey{sac.ClusterScopeKey(clusterID), sac.NamespaceScopeKey(namespace)}
	if ok, err := netpolSAC.AccessAllowed(ctx, storage.Access_READ_ACCESS, scopeKeys...); err != nil || !ok {
		return nil, err
	}

	return netPols, nil
}

func (ds *datastoreImpl) CountMatchingNetworkPolicies(ctx context.Context, clusterID, namespace string) (int, error) {
	if namespace == "" {
		netPols, err := ds.GetNetworkPolicies(ctx, clusterID, "")
		if err != nil {
			return 0, err
		}
		return len(netPols), nil
	}

	scopeKeys := []sac.ScopeKey{sac.ClusterScopeKey(clusterID), sac.NamespaceScopeKey(namespace)}
	if ok, err := netpolSAC.AccessAllowed(ctx, storage.Access_READ_ACCESS, scopeKeys...); err != nil || !ok {
		return 0, err
	}
	var count int
	err := pgutils.RetryIfPostgres(
		func() error {
			count = 0
			return ds.doForMatching(ctx, clusterID, namespace, func(np *storage.NetworkPolicy) {
				count++
			})
		},
	)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (ds *datastoreImpl) UpsertNetworkPolicy(ctx context.Context, np *storage.NetworkPolicy) error {
	if !netpolSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(np).IsAllowed() {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Upsert(ctx, np)
}

func (ds *datastoreImpl) RemoveNetworkPolicy(ctx context.Context, id string) error {
	elevatedRemoveCheckCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkPolicy),
		))
	np, found, err := ds.getNetworkPolicy(elevatedRemoveCheckCtx, id)
	if err != nil || !found {
		return err
	}

	if !netpolSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(np).IsAllowed() {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Delete(ctx, id)
}

// UndoDataStore functionality.
///////////////////////////////

func (ds *datastoreImpl) GetUndoRecord(ctx context.Context, clusterID string) (*storage.NetworkPolicyApplicationUndoRecord, bool, error) {
	if !netpolSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS, sac.ClusterScopeKey(clusterID)).IsAllowed() {
		return nil, false, nil
	}

	undoRecord, found, err := ds.undoStorage.Get(ctx, clusterID)
	if err != nil || !found {
		return nil, false, err
	}

	return undoRecord, true, nil
}

func (ds *datastoreImpl) UpsertUndoRecord(ctx context.Context, undoRecord *storage.NetworkPolicyApplicationUndoRecord) error {
	if !netpolSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS, sac.ClusterScopeKey(undoRecord.GetClusterId())).IsAllowed() {
		return sac.ErrResourceAccessDenied
	}
	ds.undoStorageLock.Lock()
	defer ds.undoStorageLock.Unlock()

	previousUndo, exists, err := ds.undoStorage.Get(ctx, undoRecord.GetClusterId())
	if err != nil {
		return err
	}
	if exists {
		if undoRecord.GetApplyTimestamp().Compare(previousUndo.GetApplyTimestamp()) < 0 {
			return fmt.Errorf("apply timestamp of record to store (%v) is older than that of existing record (%v)",
				protoconv.ConvertTimestampToTimeOrDefault(undoRecord.GetApplyTimestamp(), time.Time{}),
				protoconv.ConvertTimestampToTimeOrDefault(previousUndo.GetApplyTimestamp(), time.Time{}))
		}
	}
	return ds.undoStorage.Upsert(ctx, undoRecord)
}

// UndoDeploymentDataStore functionality.
// /////////////////////////////
func (ds *datastoreImpl) GetUndoDeploymentRecord(ctx context.Context, deploymentID string) (*storage.NetworkPolicyApplicationUndoDeploymentRecord, bool, error) {
	undoRecord, found, err := ds.undoDeploymentStorage.Get(ctx, deploymentID)
	if err != nil || !found {
		return nil, false, err
	}

	if !netpolSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(undoRecord).IsAllowed() {
		return nil, false, nil
	}

	return undoRecord, true, nil
}

func (ds *datastoreImpl) UpsertUndoDeploymentRecord(ctx context.Context, undoRecord *storage.NetworkPolicyApplicationUndoDeploymentRecord) error {
	if !netpolSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(undoRecord).IsAllowed() {
		return sac.ErrResourceAccessDenied
	}

	return ds.undoDeploymentStorage.Upsert(ctx, undoRecord)
}

func (ds *datastoreImpl) getNetworkPolicy(ctx context.Context, id string) (*storage.NetworkPolicy, bool, error) {
	netpol, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	return netpol, true, nil
}

func filterResults(ctx context.Context, resourceScopeChecker sac.ScopeChecker, results []*storage.NetworkPolicy) ([]*storage.NetworkPolicy, error) {
	var allowed []*storage.NetworkPolicy
	for _, netPol := range results {
		scopeKeys := sac.KeyForNSScopedObj(netPol)
		if resourceScopeChecker.IsAllowed(scopeKeys...) {
			allowed = append(allowed, netPol)
		}
	}
	return allowed, nil
}
