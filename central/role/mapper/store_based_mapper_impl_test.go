package mapper

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	groupMocks "github.com/stackrox/rox/central/group/datastore/mocks"
	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	userMocks "github.com/stackrox/rox/central/user/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stretchr/testify/suite"
)

const (
	fakeAuthProvider = "authProvider"
)

func TestMapper(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MapperTestSuite))
}

type MapperTestSuite struct {
	suite.Suite

	requestContext context.Context

	mockCtrl *gomock.Controller

	mockGroups *groupMocks.MockDataStore
	mockRoles  *roleMocks.MockDataStore
	mockUsers  *userMocks.MockDataStore

	mapper *storeBasedMapperImpl
}

func (s *MapperTestSuite) SetupTest() {
	s.requestContext = context.Background()

	s.mockCtrl = gomock.NewController(s.T())

	s.mockGroups = groupMocks.NewMockDataStore(s.mockCtrl)
	s.mockRoles = roleMocks.NewMockDataStore(s.mockCtrl)
	s.mockUsers = userMocks.NewMockDataStore(s.mockCtrl)

	s.mapper = &storeBasedMapperImpl{
		authProviderID: fakeAuthProvider,
		groups:         s.mockGroups,
		roles:          s.mockRoles,
		users:          s.mockUsers,
	}
}

func (s *MapperTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *MapperTestSuite) TestMapperSuccessForSingleRole() {
	// The user information we expect to be upserted.
	expectedUser := &storage.User{
		Id:             "coolguysid",
		AuthProviderId: fakeAuthProvider,
		Attributes: []*storage.UserAttribute{
			{
				Key:   "email",
				Value: "coolguy@yahoo",
			},
		},
	}
	s.mockUsers.EXPECT().Upsert(s.requestContext, expectedUser).Times(1).Return(nil)

	// Expect the user to have a group mapping for a role.
	expectedGroup := &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "coolguy@yahoo",
		},
		RoleName: "TeamAwesome",
	}
	expectedAttributes := map[string][]string{
		"email": {"coolguy@yahoo"},
	}
	s.mockGroups.
		EXPECT().
		Walk(s.requestContext, fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*storage.Group{expectedGroup}, nil)

	// Expect the role to be fetched.
	expectedRole := &storage.Role{
		Name:         "TeamAwesome",
		GlobalAccess: storage.Access_READ_ACCESS,
	}
	s.mockRoles.
		EXPECT().
		GetRole(s.requestContext, "TeamAwesome").
		Times(1).
		Return(expectedRole, nil)

	userDescriptor := &permissions.UserDescriptor{
		UserID: "coolguysid",
		Attributes: map[string][]string{
			"email": {"coolguy@yahoo"},
		},
	}
	roles, err := s.mapper.FromUserDescriptor(s.requestContext, userDescriptor)
	s.NoError(err, "mapping should have succeeded")
	s.ElementsMatch([]*storage.Role{expectedRole}, roles, "since a single role was mapped, that role should be returned")
}

func (s *MapperTestSuite) TestMapperSuccessForMultiRole() {
	// The user information we expect to be upserted.
	expectedUser := &storage.User{
		Id:             "coolguysid",
		AuthProviderId: fakeAuthProvider,
		Attributes: []*storage.UserAttribute{
			{
				Key:   "email",
				Value: "coolguy@yahoo",
			},
		},
	}
	s.mockUsers.EXPECT().Upsert(s.requestContext, expectedUser).Times(1).Return(nil)

	// Expect the user to have a two group mappings for two roles.
	expectedGroup1 := &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "coolguy@yahoo",
		},
		RoleName: "TeamAwesome",
	}
	expectedGroup2 := &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "coolguy@yahoo",
		},
		RoleName: "TeamEvenAwesomer",
	}
	expectedAttributes := map[string][]string{
		"email": {"coolguy@yahoo"},
	}
	s.mockGroups.
		EXPECT().
		Walk(s.requestContext, fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*storage.Group{expectedGroup1, expectedGroup2}, nil)

	// Expect the roles to be fetched, and make the second a superset of the first.
	expectedRole1 := &storage.Role{
		Name:         "TeamAwesome",
		GlobalAccess: storage.Access_READ_ACCESS,
	}
	expectedRole2 := &storage.Role{
		Name:         "TeamAwesome",
		GlobalAccess: storage.Access_READ_WRITE_ACCESS,
	}
	s.mockRoles.
		EXPECT().
		GetRole(s.requestContext, "TeamAwesome").
		Times(1).
		Return(expectedRole1, nil)
	s.mockRoles.
		EXPECT().
		GetRole(s.requestContext, "TeamEvenAwesomer").
		Times(1).
		Return(expectedRole2, nil)

	// Call the mapper for a user.
	userDescriptor := &permissions.UserDescriptor{
		UserID: "coolguysid",
		Attributes: map[string][]string{
			"email": {"coolguy@yahoo"},
		},
	}
	roles, err := s.mapper.FromUserDescriptor(s.requestContext, userDescriptor)
	s.Require().NoError(err, "mapping should have succeeded")

	s.ElementsMatch([]*storage.Role{expectedRole1, expectedRole2}, roles, "expected both roles to be present")
}

func (s *MapperTestSuite) TestUserUpsertFailureDoesntMatter() {
	// The user information we expect to be upserted.
	expectedUser := &storage.User{
		Id:             "coolguysid",
		AuthProviderId: fakeAuthProvider,
		Attributes: []*storage.UserAttribute{
			{
				Key:   "email",
				Value: "coolguy@yahoo",
			},
		},
	}
	s.mockUsers.EXPECT().Upsert(s.requestContext, expectedUser).Times(1).Return(errors.New("error that shouldnt matter"))

	// Expect the user to have a group mapping for a role.
	expectedGroup := &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "coolguy@yahoo",
		},
		RoleName: "TeamAwesome",
	}
	expectedAttributes := map[string][]string{
		"email": {"coolguy@yahoo"},
	}
	s.mockGroups.
		EXPECT().
		Walk(s.requestContext, fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*storage.Group{expectedGroup}, nil)

	// Expect the role to be fetched.
	expectedRole := &storage.Role{
		Name:         "TeamAwesome",
		GlobalAccess: storage.Access_READ_ACCESS,
	}
	s.mockRoles.
		EXPECT().
		GetRole(s.requestContext, "TeamAwesome").
		Times(1).
		Return(expectedRole, nil)

	// Call the mapper for a user.
	userDescriptor := &permissions.UserDescriptor{
		UserID: "coolguysid",
		Attributes: map[string][]string{
			"email": {"coolguy@yahoo"},
		},
	}
	roles, err := s.mapper.FromUserDescriptor(s.requestContext, userDescriptor)
	s.NoError(err, "mapping should have succeeded")
	s.ElementsMatch([]*storage.Role{expectedRole}, roles, "since a single role was mapped, that role should be returned")
}

func (s *MapperTestSuite) TestGroupWalkFailureCausesError() {
	// The user information we expect to be upserted.
	expectedUser := &storage.User{
		Id:             "coolguysid",
		AuthProviderId: fakeAuthProvider,
		Attributes: []*storage.UserAttribute{
			{
				Key:   "email",
				Value: "coolguy@yahoo",
			},
		},
	}
	s.mockUsers.EXPECT().Upsert(s.requestContext, expectedUser).Times(1).Return(nil)

	// Expect the user to have a group mapping for a role.
	expectedAttributes := map[string][]string{
		"email": {"coolguy@yahoo"},
	}
	s.mockGroups.
		EXPECT().
		Walk(s.requestContext, fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*storage.Group{}, errors.New("error should be returned"))

	// Call the mapper for a user.
	userDescriptor := &permissions.UserDescriptor{
		UserID: "coolguysid",
		Attributes: map[string][]string{
			"email": {"coolguy@yahoo"},
		},
	}
	_, err := s.mapper.FromUserDescriptor(s.requestContext, userDescriptor)
	s.Error(err, "mapping should have succeeded")
}

func (s *MapperTestSuite) TestRoleFetchFailureCausesError() {
	// The user information we expect to be upserted.
	expectedUser := &storage.User{
		Id:             "coolguysid",
		AuthProviderId: fakeAuthProvider,
		Attributes: []*storage.UserAttribute{
			{
				Key:   "email",
				Value: "coolguy@yahoo",
			},
		},
	}
	s.mockUsers.EXPECT().Upsert(s.requestContext, expectedUser).Times(1).Return(nil)

	// Expect the user to have a group mapping for a role.
	expectedGroup := &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "coolguy@yahoo",
		},
		RoleName: "TeamAwesome",
	}
	expectedAttributes := map[string][]string{
		"email": {"coolguy@yahoo"},
	}
	s.mockGroups.
		EXPECT().
		Walk(s.requestContext, fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*storage.Group{expectedGroup}, nil)

	// Expect the role to be fetched.
	s.mockRoles.
		EXPECT().
		GetRole(s.requestContext, "TeamAwesome").
		Times(1).
		Return(nil, errors.New("error should be returned"))

	// Call the mapper for a user.
	userDescriptor := &permissions.UserDescriptor{
		UserID: "coolguysid",
		Attributes: map[string][]string{
			"email": {"coolguy@yahoo"},
		},
	}
	_, err := s.mapper.FromUserDescriptor(s.requestContext, userDescriptor)
	s.Error(err, "mapping should have succeeded")
}
