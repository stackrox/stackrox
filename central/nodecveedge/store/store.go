package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for NodeCVEEdges.
//go:generate mockgen-wrapper
type Store interface {
	GetAll() ([]*storage.NodeCVEEdge, error)
	Count() (int, error)
	Get(id string) (*storage.NodeCVEEdge, bool, error)
	GetBatch(ids []string) ([]*storage.NodeCVEEdge, []int, error)

	Exists(id string) (bool, error)

	Upsert(edges ...*storage.NodeCVEEdge) error
	Delete(ids ...string) error
}
