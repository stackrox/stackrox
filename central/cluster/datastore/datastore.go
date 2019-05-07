package datastore

import (
	"context"
	"time"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/cluster/index"
	"github.com/stackrox/rox/central/cluster/store"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/globaldatastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error)
	GetClusters(ctx context.Context) ([]*storage.Cluster, error)
	CountClusters(ctx context.Context) (int, error)

	AddCluster(ctx context.Context, cluster *storage.Cluster) (string, error)
	UpdateCluster(ctx context.Context, cluster *storage.Cluster) error
	RemoveCluster(ctx context.Context, id string) error
	UpdateClusterContactTime(ctx context.Context, id string, t time.Time) error
	UpdateClusterStatus(ctx context.Context, id string, status *storage.ClusterStatus) error

	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
}

// New returns an instance of DataStore.
func New(
	storage store.Store,
	indexer index.Indexer,
	ads alertDataStore.DataStore,
	dds deploymentDataStore.DataStore,
	ns nodeDataStore.GlobalDataStore,
	ss secretDataStore.DataStore,
	cm connection.Manager,
	notifier notifierProcessor.Processor) (DataStore, error) {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		ads:      ads,
		dds:      dds,
		ns:       ns,
		ss:       ss,
		cm:       cm,
		notifier: notifier,
	}
	if err := ds.buildIndex(); err != nil {
		return ds, err
	}
	go ds.cleanUpNodeStore(context.TODO())
	return ds, nil
}
