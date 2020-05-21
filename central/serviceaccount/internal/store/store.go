package store

import (
	storage "github.com/stackrox/rox/generated/storage"
)

// Store encapsulates the service account store interface
type Store interface {
	Get(id string) (*storage.ServiceAccount, bool, error)
	GetMany(ids []string) ([]*storage.ServiceAccount, []int, error)
	Walk(func(sa *storage.ServiceAccount) error) error

	Upsert(serviceaccount *storage.ServiceAccount) error
	Delete(id string) error
}
