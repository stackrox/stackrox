package store

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/bolt"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore"
	undoDeploymentPostgres "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore/postgres"
	undoDeploymentRocksDB "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore/rocksdb"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore"
	undobolt "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore/bolt"
	undopostgres "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	"go.etcd.io/bbolt"
)

var (
	log = logging.LoggerForModule()
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
func New(storage store.Store, undoStorage undostore.UndoStore, undoDeploymentStorage undodeploymentstore.UndoDeploymentStore) DataStore {
	return &datastoreImpl{
		storage:               storage,
		undoStorage:           undoStorage,
		undoDeploymentStorage: undoDeploymentStorage,
	}
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	dbstore := postgres.New(pool)
	undodbstore := undopostgres.New(pool)
	undodeploymentdbstore := undoDeploymentPostgres.New(pool)
	return New(dbstore, undodbstore, undodeploymentdbstore), nil
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(_ *testing.T, rocksengine *rocksdbBase.RocksDB, boltengine *bbolt.DB) (DataStore, error) {
	dbstore := bolt.New(boltengine)
	undodbstore := undobolt.New(boltengine)
	undodeploymentdbstore, err := undoDeploymentRocksDB.New(rocksengine)
	if err != nil {
		return nil, err
	}
	return New(dbstore, undodbstore, undodeploymentdbstore), nil
}
