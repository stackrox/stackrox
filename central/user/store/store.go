package store

import (
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/expiringcache"
)

// Store is the storage and tracking mechanism for users.
//go:generate mockgen-wrapper Store
type Store interface {
	GetUser(id string) (*v1.User, error)
	GetAllUsers() ([]*v1.User, error)

	Upsert(*v1.User) error
}

// New returns a new instance of a Store.
// For now we will store information for up to 1000 users for 1 day.
func New() Store {
	return &storeImpl{
		ec: expiringcache.NewExpiringCacheOrPanic(1000, 24*time.Hour),
	}
}
