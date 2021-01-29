package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for Node Component Edges.
//go:generate mockgen-wrapper
type Store interface {
	Count() (int, error)
	Exists(id string) (bool, error)

	GetAll() ([]*storage.NodeComponentEdge, error)
	Get(id string) (*storage.NodeComponentEdge, bool, error)
	GetBatch(ids []string) ([]*storage.NodeComponentEdge, []int, error)
}
