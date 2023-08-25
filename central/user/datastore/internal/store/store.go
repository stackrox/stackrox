package store

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
)

// Store is the storage and tracking mechanism for users.
//
//go:generate mockgen-wrapper
type Store interface {
	GetUser(id string) (*storage.User, error)
	GetAllUsers() ([]*storage.User, error)

	Upsert(*storage.User) error
}

// New returns a new instance of a Store.
// The information is stored for 1 day.
func New() Store {
	return &storeImpl{
		ec: expiringcache.NewExpiringCache(24 * time.Hour),
	}
}
