package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for process indicators.
//go:generate mockgen-wrapper
type Store interface {
	Get(id string) (*storage.ProcessIndicator, bool, error)
	GetMany(ids []string) ([]*storage.ProcessIndicator, []int, error)

	UpsertMany([]*storage.ProcessIndicator) error
	DeleteMany(id []string) error

	AckKeysIndexed(keys ...string) error
	GetKeysToIndex() ([]string, error)

	Walk(func(pi *storage.ProcessIndicator) error) error
}
