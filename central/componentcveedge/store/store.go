package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for CVEs.
//go:generate mockgen-wrapper
type Store interface {
	GetAll() ([]*storage.ComponentCVEEdge, error)
	Count() (int, error)
	Get(id string) (*storage.ComponentCVEEdge, bool, error)
	GetBatch(ids []string) ([]*storage.ComponentCVEEdge, []int, error)

	Exists(id string) (bool, error)

	Upsert(cve ...*storage.ComponentCVEEdge) error
	Delete(id ...string) error
}
