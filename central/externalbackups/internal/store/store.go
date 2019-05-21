package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store implements a store of all external backups in a cluster.
//go:generate mockgen-wrapper Store
type Store interface {
	ListBackups() ([]*storage.ExternalBackup, error)
	GetBackup(id string) (*storage.ExternalBackup, error)
	UpsertBackup(backup *storage.ExternalBackup) error
	RemoveBackup(id string) error
}
