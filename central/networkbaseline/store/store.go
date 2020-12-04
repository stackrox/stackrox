package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for network baselines.
//go:generate mockgen-wrapper
type Store interface {
	Exists(id string) (bool, error)

	GetIDs() ([]string, error)
	Get(id string) (*storage.NetworkBaseline, bool, error)
	GetMany(ids []string) ([]*storage.NetworkBaseline, []int, error)

	Upsert(baseline *storage.NetworkBaseline) error
	Delete(id string) error
	DeleteMany(ids []string) error

	Walk(fn func(baseline *storage.NetworkBaseline) error) error
}
