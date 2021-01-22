package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for Node Component Edges.
//go:generate mockgen-wrapper
type Store interface {
	GetAll() ([]*storage.NodeComponentEdge, error)
	Count() (int, error)
	Get(id string) (*storage.NodeComponentEdge, bool, error)
	GetBatch(ids []string) ([]*storage.NodeComponentEdge, []int, error)

	Exists(id string) (bool, error)

	Upsert(cve ...*storage.NodeComponentEdge) error
	Delete(ids ...string) error
}
