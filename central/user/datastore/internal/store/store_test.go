package store

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/suite"
)

func TestUserStore(t *testing.T) {
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
		storage.User_builder{
			Id: "user1",
			Attributes: []*storage.UserAttribute{
				storage.UserAttribute_builder{
					Key:   "name",
					Value: "user1",
				}.Build(),
				storage.UserAttribute_builder{
					Key:   "email",
					Value: "user@derp.com",
				}.Build(),
			},
		}.Build(),
		storage.User_builder{
			Id: "user2",
			Attributes: []*storage.UserAttribute{
				storage.UserAttribute_builder{
					Key:   "name",
					Value: "user2",
				}.Build(),
			},
		}.Build(),
		storage.User_builder{
			Id: "user3",
			Attributes: []*storage.UserAttribute{
				storage.UserAttribute_builder{
					Key:   "groups",
					Value: "squad",
				}.Build(),
				storage.UserAttribute_builder{
					Key:   "name",
					Value: "user3",
				}.Build(),
			},
		}.Build(),
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
		protoassert.Equal(s.T(), a, full)
	}

	retrievedUsers, err := s.sto.GetAllUsers()
	s.NoError(err)
	protoassert.ElementsMatch(s.T(), users, retrievedUsers)
}
