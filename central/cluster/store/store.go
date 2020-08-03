package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// ClusterStore provides storage functionality for clusters.
//go:generate mockgen-wrapper
type ClusterStore interface {
	Count() (int, error)
	Walk(fn func(obj *storage.Cluster) error) error

	Get(id string) (*storage.Cluster, bool, error)
	GetMany(ids []string) ([]*storage.Cluster, []int, error)

	Upsert(cluster *storage.Cluster) error
	Delete(id string) error
}

// ClusterHealthStore provides storage functionality for cluster health.
//go:generate mockgen-wrapper
type ClusterHealthStore interface {
	Get(id string) (*storage.ClusterHealthStatus, bool, error)
	GetMany(ids []string) ([]*storage.ClusterHealthStatus, []int, error)
	UpsertWithID(id string, obj *storage.ClusterHealthStatus) error
	UpsertManyWithIDs(ids []string, objs []*storage.ClusterHealthStatus) error

	Delete(id string) error
	WalkAllWithID(fn func(id string, obj *storage.ClusterHealthStatus) error) error
}
