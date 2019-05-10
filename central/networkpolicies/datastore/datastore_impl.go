package store

import (
	"context"

	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore"
	"github.com/stackrox/rox/generated/storage"
)

type datastoreImpl struct {
	storage     store.Store
	undoStorage undostore.UndoStore
}

func (ds *datastoreImpl) GetNetworkPolicy(_ context.Context, id string) (*storage.NetworkPolicy, bool, error) {
	return ds.storage.GetNetworkPolicy(id)
}

func (ds *datastoreImpl) GetNetworkPolicies(_ context.Context, clusterID, namespace string) ([]*storage.NetworkPolicy, error) {
	return ds.storage.GetNetworkPolicies(clusterID, namespace)
}

func (ds *datastoreImpl) CountMatchingNetworkPolicies(_ context.Context, clusterID, namespace string) (int, error) {
	return ds.storage.CountMatchingNetworkPolicies(clusterID, namespace)
}

func (ds *datastoreImpl) CountNetworkPolicies(_ context.Context) (int, error) {
	return ds.storage.CountNetworkPolicies()
}

func (ds *datastoreImpl) AddNetworkPolicy(_ context.Context, np *storage.NetworkPolicy) error {
	return ds.storage.AddNetworkPolicy(np)
}

func (ds *datastoreImpl) UpdateNetworkPolicy(_ context.Context, np *storage.NetworkPolicy) error {
	return ds.storage.UpdateNetworkPolicy(np)
}

func (ds *datastoreImpl) RemoveNetworkPolicy(_ context.Context, id string) error {
	return ds.storage.RemoveNetworkPolicy(id)
}

// UndoDataStore functionality.
///////////////////////////////

func (ds *datastoreImpl) GetUndoRecord(_ context.Context, clusterID string) (*storage.NetworkPolicyApplicationUndoRecord, bool, error) {
	return ds.undoStorage.GetUndoRecord(clusterID)
}

func (ds *datastoreImpl) UpsertUndoRecord(_ context.Context, clusterID string, undoRecord *storage.NetworkPolicyApplicationUndoRecord) error {
	return ds.undoStorage.UpsertUndoRecord(clusterID, undoRecord)
}
