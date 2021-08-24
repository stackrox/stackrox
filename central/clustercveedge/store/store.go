package store

import (
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for cluster-cve edges.
//go:generate mockgen-wrapper
type Store interface {
	Count() (int, error)
	Exists(id string) (bool, error)

	GetAll() ([]*storage.ClusterCVEEdge, error)
	Get(id string) (*storage.ClusterCVEEdge, bool, error)
	GetBatch(ids []string) ([]*storage.ClusterCVEEdge, []int, error)

	Upsert(cveParts ...converter.ClusterCVEParts) error
	Delete(ids ...string) error
}
