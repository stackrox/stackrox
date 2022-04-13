package store

import (
	"github.com/stackrox/stackrox/generated/storage"
)

// Store provides storage functionality for image-component edges.
//go:generate mockgen-wrapper
type Store interface {
	Count() (int, error)
	Exists(id string) (bool, error)

	GetAll() ([]*storage.ImageComponentEdge, error)
	Get(id string) (*storage.ImageComponentEdge, bool, error)
	GetBatch(ids []string) ([]*storage.ImageComponentEdge, []int, error)
}
