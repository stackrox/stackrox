package store

import "github.com/stackrox/rox/generated/storage"

// Store provides storage functionality for network graph configuration.
//go:generate mockgen-wrapper
type Store interface {
	Get(id string) (*storage.NetworkGraphConfig, bool, error)
	UpsertWithID(id string, cluster *storage.NetworkGraphConfig) error
}
