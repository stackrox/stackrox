package store

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
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
	v, ok := s.ec.Get(id)
	if !ok {
		return nil, nil
	}
	user, _ := v.(*storage.User)
	return user, nil
}

// Upsert adds a user.
func (s *storeImpl) Upsert(user *storage.User) error {
	s.ec.Add(user.GetId(), user)
	return nil
}
