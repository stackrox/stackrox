package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	Get(id string) (*storage.NamespaceMetadata, bool, error)
	Walk(func(namespace *storage.NamespaceMetadata) error) error
	Upsert(*storage.NamespaceMetadata) error
	Delete(id string) error
}
