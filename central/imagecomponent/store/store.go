package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for Image Components.
//go:generate mockgen-wrapper
type Store interface {
	GetAll() ([]*storage.ImageComponent, error)
	Count() (int, error)
	Get(id string) (*storage.ImageComponent, bool, error)
	GetBatch(ids []string) ([]*storage.ImageComponent, []int, error)

	Exists(id string) (bool, error)

	Upsert(component *storage.ImageComponent) error
	UpsertBatch(components []*storage.ImageComponent) error

	Delete(id string) error
	DeleteBatch(ids []string) error

	GetTxnCount() (txNum uint64, err error)
	IncTxnCount() error
}
