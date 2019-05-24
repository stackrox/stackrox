package store

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	netpolSAC = sac.ForResource(resources.NetworkPolicy)
)

type datastoreImpl struct {
	storage     store.Store
	undoStorage undostore.UndoStore
}

func (ds *datastoreImpl) GetNetworkPolicy(ctx context.Context, id string) (*storage.NetworkPolicy, bool, error) {
	np, found, err := ds.getNetworkPolicy(id)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := netpolSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).ForNamespaceScopedObject(np).Allowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	return np, true, nil
}

func (ds *datastoreImpl) GetNetworkPolicies(ctx context.Context, clusterID, namespace string) ([]*storage.NetworkPolicy, error) {
	netPols, err := ds.storage.GetNetworkPolicies(clusterID, namespace)
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
	return ds.storage.CountMatchingNetworkPolicies(clusterID, namespace)
}

func (ds *datastoreImpl) AddNetworkPolicy(ctx context.Context, np *storage.NetworkPolicy) error {
	if ok, err := netpolSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(np).Allowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.AddNetworkPolicy(np)
}

func (ds *datastoreImpl) UpdateNetworkPolicy(ctx context.Context, np *storage.NetworkPolicy) error {
	if ok, err := netpolSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(np).Allowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.UpdateNetworkPolicy(np)
}

func (ds *datastoreImpl) RemoveNetworkPolicy(ctx context.Context, id string) error {
	np, found, err := ds.getNetworkPolicy(id)
	if err != nil || !found {
		return err
	}

	if ok, err := netpolSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).ForNamespaceScopedObject(np).Allowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.RemoveNetworkPolicy(id)
}

// UndoDataStore functionality.
///////////////////////////////

func (ds *datastoreImpl) GetUndoRecord(ctx context.Context, clusterID string) (*storage.NetworkPolicyApplicationUndoRecord, bool, error) {
	if ok, err := netpolSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS, sac.ClusterScopeKey(clusterID)).Allowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	undoRecord, found, err := ds.undoStorage.GetUndoRecord(clusterID)
	if err != nil || !found {
		return nil, false, err
	}

	return undoRecord, true, nil
}

func (ds *datastoreImpl) UpsertUndoRecord(ctx context.Context, clusterID string, undoRecord *storage.NetworkPolicyApplicationUndoRecord) error {
	if ok, err := netpolSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS, sac.ClusterScopeKey(clusterID)).Allowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.undoStorage.UpsertUndoRecord(clusterID, undoRecord)
}

func (ds *datastoreImpl) getNetworkPolicy(id string) (*storage.NetworkPolicy, bool, error) {
	netpol, found, err := ds.storage.GetNetworkPolicy(id)
	if err != nil || !found {
		return nil, false, err
	}

	return netpol, true, nil
}

func filterResultsOnce(resourceScopeChecker sac.ScopeChecker, results []*storage.NetworkPolicy) (allowed []*storage.NetworkPolicy, maybe []*storage.NetworkPolicy) {
	for _, netPol := range results {
		scopeKeys := sac.KeyForNSScopedObj(netPol)
		if res := resourceScopeChecker.TryAllowed(scopeKeys...); res == sac.Allow {
			allowed = append(allowed, netPol)
		} else if res == sac.Unknown {
			maybe = append(maybe, netPol)
		}
	}
	return
}

func filterResults(ctx context.Context, resourceScopeChecker sac.ScopeChecker, results []*storage.NetworkPolicy) ([]*storage.NetworkPolicy, error) {
	allowed, maybe := filterResultsOnce(resourceScopeChecker, results)
	if len(maybe) > 0 {
		if err := resourceScopeChecker.PerformChecks(ctx); err != nil {
			return nil, err
		}
		extraAllowed, maybe := filterResultsOnce(resourceScopeChecker, maybe)
		if len(maybe) > 0 {
			errorhelpers.PanicOnDevelopmentf("still %d maybe results after PerformChecks", len(maybe))
		}
		allowed = append(allowed, extraAllowed...)
	}

	return allowed, nil
}
