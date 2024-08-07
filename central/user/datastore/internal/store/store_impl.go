package store

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
)

type storeImpl struct {
	ec expiringcache.Cache[string, *storage.User]
}

// GetAllUsers retrieves all users from the store.
func (s *storeImpl) GetAllUsers() ([]*storage.User, error) {
	return s.ec.GetAll(), nil
}

// GetUser retrieves a user from the store by id.
func (s *storeImpl) GetUser(id string) (*storage.User, error) {
	user, _ := s.ec.Get(id)
	return user, nil
}

// Upsert adds a user.
func (s *storeImpl) Upsert(user *storage.User) error {
	s.ec.Add(user.GetId(), user)
	return nil
}
