package store

import (
	"context"

	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// UndoDataStore provides storage functionality for the undo records resulting from policy application.
//go:generate mockgen-wrapper
type UndoDataStore interface {
	GetUndoRecord(ctx context.Context, clusterID string) (*storage.NetworkPolicyApplicationUndoRecord, bool, error)
	UpsertUndoRecord(ctx context.Context, clusterID string, undoRecord *storage.NetworkPolicyApplicationUndoRecord) error
}

// DataStore provides storage functionality for network policies.
//go:generate mockgen-wrapper
type DataStore interface {
	GetNetworkPolicy(ctx context.Context, id string) (*storage.NetworkPolicy, bool, error)
	GetNetworkPolicies(ctx context.Context, clusterID, namespace string) ([]*storage.NetworkPolicy, error)
	CountMatchingNetworkPolicies(ctx context.Context, clusterID, namespace string) (int, error)

	UpsertNetworkPolicy(ctx context.Context, np *storage.NetworkPolicy) error
	RemoveNetworkPolicy(ctx context.Context, id string) error

	UndoDataStore
	UndoDeploymentDataStore
}

// UndoDeploymentDataStore provides storage functionality for the undo deployment records resulting
// from policy application.
//go:generate mockgen-wrapper
type UndoDeploymentDataStore interface {
	GetUndoDeploymentRecord(ctx context.Context, deploymentID string) (*storage.NetworkPolicyApplicationUndoDeploymentRecord, bool, error)
	UpsertUndoDeploymentRecord(ctx context.Context, undoRecord *storage.NetworkPolicyApplicationUndoDeploymentRecord) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(storage store.Store, undoStorage undostore.UndoStore, undoDeploymentStorage undodeploymentstore.UndoDeploymentStore) DataStore {
	return &datastoreImpl{
		storage:               storage,
		undoStorage:           undoStorage,
		undoDeploymentStorage: undoDeploymentStorage,
	}
}
