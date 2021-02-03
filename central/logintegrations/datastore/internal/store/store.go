package store

import "github.com/stackrox/rox/generated/storage"

// Store provides storage functionality for log integrations.
//go:generate mockgen-wrapper
type Store interface {
	Walk(fn func(obj *storage.LogIntegration) error) error
	Get(id string) (*storage.LogIntegration, bool, error)
	GetMany(ids []string) ([]*storage.LogIntegration, []int, error)
	Upsert(cluster *storage.LogIntegration) error
	Delete(id string) error
}
