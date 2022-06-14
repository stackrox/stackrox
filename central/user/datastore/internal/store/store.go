package store

import (
	"time"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/expiringcache"
)

// Store is the storage and tracking mechanism for users.
//go:generate mockgen-wrapper
type Store interface {
	GetUser(id string) (*storage.User, error)
	GetAllUsers() ([]*storage.User, error)

	Upsert(*storage.User) error
}

// New returns a new instance of a Store.
// For now we will store information for up to 1000 users for 1 day.
func New() Store {
	return &storeImpl{
		ec: expiringcache.NewExpiringCache(24 * time.Hour),
	}
}
