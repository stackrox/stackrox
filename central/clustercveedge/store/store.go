package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for CVEs.
//go:generate mockgen-wrapper
type Store interface {
	GetAll() ([]*storage.ClusterCVEEdge, error)
	Count() (int, error)
	Get(id string) (*storage.ClusterCVEEdge, bool, error)
	GetBatch(ids []string) ([]*storage.ClusterCVEEdge, []int, error)

	Exists(id string) (bool, error)

	Upsert(cve ...*storage.ClusterCVEEdge) error
	Delete(id ...string) error
}
