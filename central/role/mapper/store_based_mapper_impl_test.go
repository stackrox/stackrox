package mapper

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	groupMocks "github.com/stackrox/rox/central/group/datastore/mocks"
	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	userMocks "github.com/stackrox/rox/central/user/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stretchr/testify/suite"
)

const (
	fakeAuthProvider = "authProvider"
)

func TestAlertService(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(MapperTestSuite))
}

type MapperTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	mockGroups *groupMocks.MockDataStore
	mockRoles  *roleMocks.MockDataStore
	mockUsers  *userMocks.MockDataStore

	mapper *storeBasedMapperImpl
}

func (s *MapperTestSuite) SetupTest() {
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
	s.mockUsers.EXPECT().Upsert(context.TODO(), expectedUser).Times(1).Return(nil)

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
		Walk(context.TODO(), fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*storage.Group{expectedGroup}, nil)

	// Expect the role to be fetched.
	expectedRole := &storage.Role{
		Name:         "TeamAwesome",
		GlobalAccess: storage.Access_READ_ACCESS,
	}
	s.mockRoles.
		EXPECT().
		GetRole(context.TODO(), "TeamAwesome").
		Times(1).
		Return(expectedRole, nil)

	// Call the mapper for a user.
	tokenClaims := &tokens.Claims{
		RoxClaims: tokens.RoxClaims{
			ExternalUser: &tokens.ExternalUserClaim{
				UserID: "coolguysid",
				Attributes: map[string][]string{
					"email": {"coolguy@yahoo"},
				},
			},
		},
	}
	role, err := s.mapper.FromTokenClaims(context.TODO(), tokenClaims)
	s.NoError(err, "mapping should have succeeded")
	s.Equal(expectedRole, role, "since a single role was mapped, that role should be returned")
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
	s.mockUsers.EXPECT().Upsert(context.TODO(), expectedUser).Times(1).Return(nil)

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
		Walk(context.TODO(), fakeAuthProvider, expectedAttributes).
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
		GetRole(context.TODO(), "TeamAwesome").
		Times(1).
		Return(expectedRole1, nil)
	s.mockRoles.
		EXPECT().
		GetRole(context.TODO(), "TeamEvenAwesomer").
		Times(1).
		Return(expectedRole2, nil)

	// Call the mapper for a user.
	tokenClaims := &tokens.Claims{
		RoxClaims: tokens.RoxClaims{
			ExternalUser: &tokens.ExternalUserClaim{
				UserID: "coolguysid",
				Attributes: map[string][]string{
					"email": {"coolguy@yahoo"},
				},
			},
		},
	}
	role, err := s.mapper.FromTokenClaims(context.TODO(), tokenClaims)
	s.NoError(err, "mapping should have succeeded")

	// Permissions should be the two roles' permissions combined.
	unionRole := &storage.Role{
		GlobalAccess: storage.Access_READ_WRITE_ACCESS,
	}
	s.Equal(unionRole, role, "since a single role was mapped, that role should be returned")
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
	s.mockUsers.EXPECT().Upsert(context.TODO(), expectedUser).Times(1).Return(fmt.Errorf("error that shouldnt matter"))

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
		Walk(context.TODO(), fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*storage.Group{expectedGroup}, nil)

	// Expect the role to be fetched.
	expectedRole := &storage.Role{
		Name:         "TeamAwesome",
		GlobalAccess: storage.Access_READ_ACCESS,
	}
	s.mockRoles.
		EXPECT().
		GetRole(context.TODO(), "TeamAwesome").
		Times(1).
		Return(expectedRole, nil)

	// Call the mapper for a user.
	tokenClaims := &tokens.Claims{
		RoxClaims: tokens.RoxClaims{
			ExternalUser: &tokens.ExternalUserClaim{
				UserID: "coolguysid",
				Attributes: map[string][]string{
					"email": {"coolguy@yahoo"},
				},
			},
		},
	}
	role, err := s.mapper.FromTokenClaims(context.TODO(), tokenClaims)
	s.NoError(err, "mapping should have succeeded")
	s.Equal(expectedRole, role, "since a single role was mapped, that role should be returned")
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
	s.mockUsers.EXPECT().Upsert(context.TODO(), expectedUser).Times(1).Return(nil)

	// Expect the user to have a group mapping for a role.
	expectedAttributes := map[string][]string{
		"email": {"coolguy@yahoo"},
	}
	s.mockGroups.
		EXPECT().
		Walk(context.TODO(), fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*storage.Group{}, fmt.Errorf("error should be returned"))

	// Call the mapper for a user.
	tokenClaims := &tokens.Claims{
		RoxClaims: tokens.RoxClaims{
			ExternalUser: &tokens.ExternalUserClaim{
				UserID: "coolguysid",
				Attributes: map[string][]string{
					"email": {"coolguy@yahoo"},
				},
			},
		},
	}
	_, err := s.mapper.FromTokenClaims(context.TODO(), tokenClaims)
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
	s.mockUsers.EXPECT().Upsert(context.TODO(), expectedUser).Times(1).Return(nil)

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
		Walk(context.TODO(), fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*storage.Group{expectedGroup}, nil)

	// Expect the role to be fetched.
	s.mockRoles.
		EXPECT().
		GetRole(context.TODO(), "TeamAwesome").
		Times(1).
		Return(nil, fmt.Errorf("error should be returned"))

	// Call the mapper for a user.
	tokenClaims := &tokens.Claims{
		RoxClaims: tokens.RoxClaims{
			ExternalUser: &tokens.ExternalUserClaim{
				UserID: "coolguysid",
				Attributes: map[string][]string{
					"email": {"coolguy@yahoo"},
				},
			},
		},
	}
	_, err := s.mapper.FromTokenClaims(context.TODO(), tokenClaims)
	s.Error(err, "mapping should have succeeded")
}
