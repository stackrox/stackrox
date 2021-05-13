package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for pods.
//go:generate mockgen-wrapper
type Store interface {
	GetIDs() ([]string, error)

	Get(id string) (*storage.Pod, bool, error)
	GetMany(ids []string) ([]*storage.Pod, []int, error)
	Walk(fn func(obj *storage.Pod) error) error

	Upsert(pod *storage.Pod) error
	Delete(id string) error

	AckKeysIndexed(keys ...string) error
	GetKeysToIndex() ([]string, error)
}
