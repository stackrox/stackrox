package mapper

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	groupMocks "github.com/stackrox/rox/central/group/store/mocks"
	roleMocks "github.com/stackrox/rox/central/role/store/mocks"
	userMocks "github.com/stackrox/rox/central/user/store/mocks"
	"github.com/stackrox/rox/generated/api/v1"
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

	groupStoreMock *groupMocks.MockStore
	roleStoreMock  *roleMocks.MockStore
	userStoreMock  *userMocks.MockStore

	mapper *mapperImpl
}

func (s *MapperTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.groupStoreMock = groupMocks.NewMockStore(s.mockCtrl)
	s.roleStoreMock = roleMocks.NewMockStore(s.mockCtrl)
	s.userStoreMock = userMocks.NewMockStore(s.mockCtrl)

	s.mapper = &mapperImpl{
		authProviderID: fakeAuthProvider,
		groupStore:     s.groupStoreMock,
		roleStore:      s.roleStoreMock,
		userStore:      s.userStoreMock,
	}
}

func (s *MapperTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *MapperTestSuite) TestMapperSuccessForSingleRole() {
	// The user information we expect to be upserted.
	expectedUser := &v1.User{
		Id:             "coolguysid",
		AuthProviderId: fakeAuthProvider,
		Attributes: []*v1.UserAttribute{
			{
				Key:   "email",
				Value: "coolguy@yahoo",
			},
		},
	}
	s.userStoreMock.EXPECT().Upsert(expectedUser).Times(1).Return(nil)

	// Expect the user to have a group mapping for a role.
	expectedGroup := &v1.Group{
		Props: &v1.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "coolguy@yahoo",
		},
		RoleName: "TeamAwesome",
	}
	expectedAttributes := map[string][]string{
		"email": {"coolguy@yahoo"},
	}
	s.groupStoreMock.
		EXPECT().
		Walk(fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*v1.Group{expectedGroup}, nil)

	// Expect the role to be fetched.
	expectedRole := &v1.Role{
		Name:         "TeamAwesome",
		GlobalAccess: v1.Access_READ_ACCESS,
	}
	s.roleStoreMock.
		EXPECT().
		GetRolesBatch([]string{"TeamAwesome"}).
		Times(1).
		Return([]*v1.Role{expectedRole}, nil)

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
	role, err := s.mapper.FromTokenClaims(tokenClaims)
	s.NoError(err, "mapping should have succeeded")
	s.Equal(expectedRole, role, "since a single role was mapped, that role should be returned")
}

func (s *MapperTestSuite) TestMapperSuccessForMultiRole() {
	// The user information we expect to be upserted.
	expectedUser := &v1.User{
		Id:             "coolguysid",
		AuthProviderId: fakeAuthProvider,
		Attributes: []*v1.UserAttribute{
			{
				Key:   "email",
				Value: "coolguy@yahoo",
			},
		},
	}
	s.userStoreMock.EXPECT().Upsert(expectedUser).Times(1).Return(nil)

	// Expect the user to have a two group mappings for two roles.
	expectedGroup1 := &v1.Group{
		Props: &v1.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "coolguy@yahoo",
		},
		RoleName: "TeamAwesome",
	}
	expectedGroup2 := &v1.Group{
		Props: &v1.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "coolguy@yahoo",
		},
		RoleName: "TeamEvenAwesomer",
	}
	expectedAttributes := map[string][]string{
		"email": {"coolguy@yahoo"},
	}
	s.groupStoreMock.
		EXPECT().
		Walk(fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*v1.Group{expectedGroup1, expectedGroup2}, nil)

	// Expect the roles to be fetched, and make the second a superset of the first.
	expectedRole1 := &v1.Role{
		Name:         "TeamAwesome",
		GlobalAccess: v1.Access_READ_ACCESS,
	}
	expectedRole2 := &v1.Role{
		Name:         "TeamAwesome",
		GlobalAccess: v1.Access_READ_WRITE_ACCESS,
	}
	s.roleStoreMock.
		EXPECT().
		GetRolesBatch(gomock.Any()).
		Times(1).
		Return([]*v1.Role{expectedRole1, expectedRole2}, nil)

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
	role, err := s.mapper.FromTokenClaims(tokenClaims)
	s.NoError(err, "mapping should have succeeded")

	// Permissions should be the two roles' permissions combined.
	unionRole := &v1.Role{
		GlobalAccess: v1.Access_READ_WRITE_ACCESS,
	}
	s.Equal(unionRole, role, "since a single role was mapped, that role should be returned")
}

func (s *MapperTestSuite) TestUserUpsertFailureDoesntMatter() {
	// The user information we expect to be upserted.
	expectedUser := &v1.User{
		Id:             "coolguysid",
		AuthProviderId: fakeAuthProvider,
		Attributes: []*v1.UserAttribute{
			{
				Key:   "email",
				Value: "coolguy@yahoo",
			},
		},
	}
	s.userStoreMock.EXPECT().Upsert(expectedUser).Times(1).Return(fmt.Errorf("error that shouldnt matter"))

	// Expect the user to have a group mapping for a role.
	expectedGroup := &v1.Group{
		Props: &v1.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "coolguy@yahoo",
		},
		RoleName: "TeamAwesome",
	}
	expectedAttributes := map[string][]string{
		"email": {"coolguy@yahoo"},
	}
	s.groupStoreMock.
		EXPECT().
		Walk(fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*v1.Group{expectedGroup}, nil)

	// Expect the role to be fetched.
	expectedRole := &v1.Role{
		Name:         "TeamAwesome",
		GlobalAccess: v1.Access_READ_ACCESS,
	}
	s.roleStoreMock.
		EXPECT().
		GetRolesBatch([]string{"TeamAwesome"}).
		Times(1).
		Return([]*v1.Role{expectedRole}, nil)

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
	role, err := s.mapper.FromTokenClaims(tokenClaims)
	s.NoError(err, "mapping should have succeeded")
	s.Equal(expectedRole, role, "since a single role was mapped, that role should be returned")
}

func (s *MapperTestSuite) TestGroupWalkFailureCausesError() {
	// The user information we expect to be upserted.
	expectedUser := &v1.User{
		Id:             "coolguysid",
		AuthProviderId: fakeAuthProvider,
		Attributes: []*v1.UserAttribute{
			{
				Key:   "email",
				Value: "coolguy@yahoo",
			},
		},
	}
	s.userStoreMock.EXPECT().Upsert(expectedUser).Times(1).Return(nil)

	// Expect the user to have a group mapping for a role.
	expectedAttributes := map[string][]string{
		"email": {"coolguy@yahoo"},
	}
	s.groupStoreMock.
		EXPECT().
		Walk(fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*v1.Group{}, fmt.Errorf("error should be returned"))

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
	_, err := s.mapper.FromTokenClaims(tokenClaims)
	s.Error(err, "mapping should have succeeded")
}

func (s *MapperTestSuite) TestRoleFetchFailureCausesError() {
	// The user information we expect to be upserted.
	expectedUser := &v1.User{
		Id:             "coolguysid",
		AuthProviderId: fakeAuthProvider,
		Attributes: []*v1.UserAttribute{
			{
				Key:   "email",
				Value: "coolguy@yahoo",
			},
		},
	}
	s.userStoreMock.EXPECT().Upsert(expectedUser).Times(1).Return(nil)

	// Expect the user to have a group mapping for a role.
	expectedGroup := &v1.Group{
		Props: &v1.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "coolguy@yahoo",
		},
		RoleName: "TeamAwesome",
	}
	expectedAttributes := map[string][]string{
		"email": {"coolguy@yahoo"},
	}
	s.groupStoreMock.
		EXPECT().
		Walk(fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*v1.Group{expectedGroup}, nil)

	// Expect the role to be fetched.
	s.roleStoreMock.
		EXPECT().
		GetRolesBatch([]string{"TeamAwesome"}).
		Times(1).
		Return([]*v1.Role{}, fmt.Errorf("error should be returned"))

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
	_, err := s.mapper.FromTokenClaims(tokenClaims)
	s.Error(err, "mapping should have succeeded")
}
