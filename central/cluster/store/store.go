package store

import (
	"time"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const (
	clusterBucket                = "clusters"
	clusterLastContactTimeBucket = "clusters_last_contact"
)

// Store provides storage functionality for alerts.
//go:generate mockery -name=Store
type Store interface {
	GetCluster(id string) (*v1.Cluster, bool, error)
	GetClusters() ([]*v1.Cluster, error)
	CountClusters() (int, error)
	AddCluster(cluster *v1.Cluster) (string, error)
	UpdateCluster(cluster *v1.Cluster) error
	RemoveCluster(id string) error
	UpdateClusterContactTime(id string, t time.Time) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, clusterBucket)
	bolthelper.RegisterBucketOrPanic(db, clusterLastContactTimeBucket)
	return &storeImpl{
		DB: db,
	}
}
