package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for pods.
//go:generate mockgen-wrapper
type Store interface {
	GetPod(id string) (*storage.Pod, bool, error)
	GetPods() ([]*storage.Pod, error)
	GetPodsWithIDs(ids ...string) ([]*storage.Pod, []int, error)

	CountPods() (int, error)
	UpsertPod(pod *storage.Pod) error
	RemovePod(id string) error

	AckKeysIndexed(keys ...string) error
	GetKeysToIndex() ([]string, error)

	GetPodIDs() ([]string, error)
}
