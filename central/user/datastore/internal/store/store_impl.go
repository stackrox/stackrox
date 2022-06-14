package store

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/expiringcache"
)

type storeImpl struct {
	ec expiringcache.Cache
}

// GetAllUsers retrieves all users from the store.
func (s *storeImpl) GetAllUsers() ([]*storage.User, error) {
	msgs := s.ec.GetAll()
	if len(msgs) == 0 {
		return nil, nil
	}
	// Cast as list of users.
	users := make([]*storage.User, 0, len(msgs))
	for _, msg := range msgs {
		users = append(users, msg.(*storage.User))
	}
	return users, nil
}

// GetUser retrieves a user from the store by id.
func (s *storeImpl) GetUser(id string) (*storage.User, error) {
	user, _ := s.ec.Get(id).(*storage.User)
	return user, nil
}

// Upsert adds a user.
func (s *storeImpl) Upsert(user *storage.User) error {
	s.ec.Add(user.GetId(), user)
	return nil
}
