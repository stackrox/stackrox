package store

import (
	"github.com/stackrox/rox/generated/storage"
)

// Store provides access and update functions for secrets.
//go:generate mockgen-wrapper
type Store interface {
	Count() (int, error)
	Get(id string) (*storage.Secret, bool, error)
	GetMany(ids []string) ([]*storage.Secret, []int, error)
	Walk(func(secret *storage.Secret) error) error

	Upsert(secret *storage.Secret) error
	Delete(id string) error
}
