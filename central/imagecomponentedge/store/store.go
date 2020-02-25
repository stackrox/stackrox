package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for CVEs.
//go:generate mockgen-wrapper
type Store interface {
	GetAll() ([]*storage.ImageComponentEdge, error)
	Count() (int, error)
	Get(id string) (*storage.ImageComponentEdge, bool, error)
	GetBatch(ids []string) ([]*storage.ImageComponentEdge, []int, error)

	Exists(id string) (bool, error)

	Upsert(cve ...*storage.ImageComponentEdge) error
	Delete(ids ...string) error
}
