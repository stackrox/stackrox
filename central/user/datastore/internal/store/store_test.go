package store

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestUserStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UserStoreTestSuite))
}

type UserStoreTestSuite struct {
	suite.Suite

	sto Store
}

func (s *UserStoreTestSuite) SetupSuite() {
	s.sto = New()
}

func (s *UserStoreTestSuite) TestUserStore() {
	users := []*storage.User{
		{
			Id: "user1",
			Attributes: []*storage.UserAttribute{
				{
					Key:   "name",
					Value: "user1",
				},
				{
					Key:   "email",
					Value: "user@derp.com",
				},
			},
		},
		{
			Id: "user2",
			Attributes: []*storage.UserAttribute{
				{
					Key:   "name",
					Value: "user2",
				},
			},
		},
		{
			Id: "user3",
			Attributes: []*storage.UserAttribute{
				{
					Key:   "groups",
					Value: "squad",
				},
				{
					Key:   "name",
					Value: "user3",
				},
			},
		},
	}

	for _, a := range users {
		s.NoError(s.sto.Upsert(a))
	}

	for _, a := range users {
		s.NoError(s.sto.Upsert(a))
	}

	for _, a := range users {
		full, err := s.sto.GetUser(a.GetId())
		s.NoError(err)
		s.Equal(a, full)
	}

	retrievedUsers, err := s.sto.GetAllUsers()
	s.NoError(err)
	s.ElementsMatch(users, retrievedUsers)
}
