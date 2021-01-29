package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for NodeCVEEdges.
//go:generate mockgen-wrapper
type Store interface {
	Count() (int, error)
	Exists(id string) (bool, error)

	GetAll() ([]*storage.NodeCVEEdge, error)
	Get(id string) (*storage.NodeCVEEdge, bool, error)
	GetBatch(ids []string) ([]*storage.NodeCVEEdge, []int, error)
}
