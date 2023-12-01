package store

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/search"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore"
	undoDeploymentPostgres "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore/postgres"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore"
	undopostgres "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// UndoDataStore provides storage functionality for the undo records resulting from policy application.
//
//go:generate mockgen-wrapper
type UndoDataStore interface {
	GetUndoRecord(ctx context.Context, clusterID string) (*storage.NetworkPolicyApplicationUndoRecord, bool, error)
	UpsertUndoRecord(ctx context.Context, undoRecord *storage.NetworkPolicyApplicationUndoRecord) error
}

// DataStore provides storage functionality for network policies.
//
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
//
//go:generate mockgen-wrapper
type UndoDeploymentDataStore interface {
	GetUndoDeploymentRecord(ctx context.Context, deploymentID string) (*storage.NetworkPolicyApplicationUndoDeploymentRecord, bool, error)
	UpsertUndoDeploymentRecord(ctx context.Context, undoRecord *storage.NetworkPolicyApplicationUndoDeploymentRecord) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(storage store.Store, searcher search.Searcher, undoStorage undostore.UndoStore, undoDeploymentStorage undodeploymentstore.UndoDeploymentStore) DataStore {
	return &datastoreImpl{
		storage:               storage,
		searcher:              searcher,
		undoStorage:           undoStorage,
		undoDeploymentStorage: undoDeploymentStorage,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	dbstore := pgStore.New(pool)
	searcher := search.New(pgStore.NewIndexer(pool))
	undodbstore := undopostgres.New(pool)
	undodeploymentdbstore := undoDeploymentPostgres.New(pool)
	return New(dbstore, searcher, undodbstore, undodeploymentdbstore), nil
}

// GetBenchPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetBenchPostgresDataStore(_ testing.TB, pool postgres.DB) (DataStore, error) {
	dbstore := pgStore.New(pool)
	undodbstore := undopostgres.New(pool)
	undodeploymentdbstore := undoDeploymentPostgres.New(pool)
	return New(dbstore, nil, undodbstore, undodeploymentdbstore), nil
}
