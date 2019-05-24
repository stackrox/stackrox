package store

import (
	"context"

	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// UndoDataStore provides storage functionality for the undo records resulting from policy application.
//go:generate mockgen-wrapper UndoDataStore
type UndoDataStore interface {
	GetUndoRecord(ctx context.Context, clusterID string) (*storage.NetworkPolicyApplicationUndoRecord, bool, error)
	UpsertUndoRecord(ctx context.Context, clusterID string, undoRecord *storage.NetworkPolicyApplicationUndoRecord) error
}

// DataStore provides storage functionality for network policies.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	GetNetworkPolicy(ctx context.Context, id string) (*storage.NetworkPolicy, bool, error)
	GetNetworkPolicies(ctx context.Context, clusterID, namespace string) ([]*storage.NetworkPolicy, error)
	CountMatchingNetworkPolicies(ctx context.Context, clusterID, namespace string) (int, error)

	AddNetworkPolicy(ctx context.Context, np *storage.NetworkPolicy) error
	UpdateNetworkPolicy(ctx context.Context, np *storage.NetworkPolicy) error
	RemoveNetworkPolicy(ctx context.Context, id string) error

	UndoDataStore
}

// New returns a new Store instance using the provided bolt DB instance.
func New(storage store.Store, undoStorage undostore.UndoStore) DataStore {
	return &datastoreImpl{
		storage:     storage,
		undoStorage: undoStorage,
	}
}
