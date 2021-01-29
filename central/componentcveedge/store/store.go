package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for component-cve edges.
//go:generate mockgen-wrapper
type Store interface {
	Count() (int, error)
	Exists(id string) (bool, error)

	GetAll() ([]*storage.ComponentCVEEdge, error)
	Get(id string) (*storage.ComponentCVEEdge, bool, error)
	GetBatch(ids []string) ([]*storage.ComponentCVEEdge, []int, error)
}
