package store

import (
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var (
	clusterBucket                = []byte("clusters")
	clusterLastContactTimeBucket = []byte("clusters_last_contact")
	clusterStatusBucket          = []byte("cluster_status")
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper Store
type Store interface {
	GetCluster(id string) (*storage.Cluster, bool, error)
	GetClusters() ([]*storage.Cluster, error)
	CountClusters() (int, error)
	AddCluster(cluster *storage.Cluster) (string, error)
	UpdateCluster(cluster *storage.Cluster) error
	RemoveCluster(id string) error
	UpdateClusterContactTime(id string, t time.Time) error
	UpdateClusterStatus(id string, status *storage.ClusterStatus) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, clusterBucket)
	bolthelper.RegisterBucketOrPanic(db, clusterLastContactTimeBucket)
	bolthelper.RegisterBucketOrPanic(db, clusterStatusBucket)
	return &storeImpl{
		DB: db,
	}
}
