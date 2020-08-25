package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for ImageCVEEdges.
//go:generate mockgen-wrapper
type Store interface {
	GetAll() ([]*storage.ImageCVEEdge, error)
	Count() (int, error)
	Get(id string) (*storage.ImageCVEEdge, bool, error)
	GetBatch(ids []string) ([]*storage.ImageCVEEdge, []int, error)

	Exists(id string) (bool, error)

	Upsert(edges ...*storage.ImageCVEEdge) error
	Delete(ids ...string) error
}
