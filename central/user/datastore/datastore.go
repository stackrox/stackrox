package datastore

import (
	"context"

	"github.com/stackrox/rox/central/user/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the datastore for users.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetUser(ctx context.Context, name string) (*storage.User, error)
	GetAllUsers(ctx context.Context) ([]*storage.User, error)

	Upsert(ctx context.Context, user *storage.User) error
}

// New returns a new DataStore instance.
func New(storage store.Store) DataStore {
	return &dataStoreImpl{
		storage: storage,
	}
}
