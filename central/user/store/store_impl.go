package store

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/expiringcache"
)

type storeImpl struct {
	ec expiringcache.Cache
}

// GetAllUsers retrieves all users from the store.
func (s *storeImpl) GetAllUsers() ([]*v1.User, error) {
	msgs := s.ec.GetAll()
	if len(msgs) == 0 {
		return nil, nil
	}
	// Cast as list of users.
	users := make([]*v1.User, 0, len(msgs))
	for _, msg := range msgs {
		users = append(users, msg.(*v1.User))
	}
	return users, nil
}

// GetUser retrieves a user from the store by id.
func (s *storeImpl) GetUser(id string) (*v1.User, error) {
	user, _ := s.ec.Get(id).(*v1.User)
	return user, nil
}

// Upsert adds a user.
func (s *storeImpl) Upsert(user *v1.User) error {
	s.ec.Add(user.GetId(), user)
	return nil
}
