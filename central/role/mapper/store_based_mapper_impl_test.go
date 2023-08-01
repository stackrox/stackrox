package mapper

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	groupMocks "github.com/stackrox/rox/central/group/datastore/mocks"
	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	userMocks "github.com/stackrox/rox/central/user/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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

func (s *MapperTestSuite) TestMapperSuccessForRoleAbsence() {
	// The user information.
	expectedUser := &storage.User{
		Id:             "testuserid",
		AuthProviderId: fakeAuthProvider,
		Attributes: []*storage.UserAttribute{
			{
				Key:   "email",
				Value: "testuser@domain.tld",
			},
		},
	}
	s.mockUsers.EXPECT().GetUser(s.requestContext, expectedUser.Id).Times(1).Return(nil, nil)
	s.mockUsers.EXPECT().Upsert(s.requestContext, expectedUser).Times(1).Return(nil)

	expectedAttributes := map[string][]string{
		"email": {"testuser@domain.tld"},
	}
	// Expect the user to have no group mapping.
	s.mockGroups.EXPECT().Walk(s.requestContext, fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*storage.Group{}, nil)

	userDescriptor := &permissions.UserDescriptor{
		UserID: "testuserid",
		Attributes: map[string][]string{
			"email": {"testuser@domain.tld"},
		},
	}

	roles, err := s.mapper.FromUserDescriptor(s.requestContext, userDescriptor)
	s.NoError(err, "mapping should have succeeded")
	s.ElementsMatch(nil, roles, "since no role was mapped, no role should be returned")
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
	s.mockUsers.EXPECT().GetUser(s.requestContext, expectedUser.Id).Times(1).Return(nil, nil)
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
	expectedResolvedRole := roletest.NewResolvedRoleWithDenyAll(
		"TeamAwesome",
		utils.FromResourcesWithAccess(resources.AllResourcesViewPermissions()...))
	s.mockRoles.
		EXPECT().
		GetAndResolveRole(s.requestContext, "TeamAwesome").
		Times(1).
		Return(expectedResolvedRole, nil)

	userDescriptor := &permissions.UserDescriptor{
		UserID: "coolguysid",
		Attributes: map[string][]string{
			"email": {"coolguy@yahoo"},
		},
	}
	roles, err := s.mapper.FromUserDescriptor(s.requestContext, userDescriptor)
	s.NoError(err, "mapping should have succeeded")
	s.ElementsMatch([]permissions.ResolvedRole{expectedResolvedRole}, roles, "since a single role was mapped, that role should be returned")
}

func (s *MapperTestSuite) TestMapperSuccessForOnlyNoneRole() {
	// The user information we expect to be upserted.
	expectedUser := &storage.User{
		Id:             "testuserid",
		AuthProviderId: fakeAuthProvider,
		Attributes: []*storage.UserAttribute{
			{
				Key:   "email",
				Value: "testuser@domain.tld",
			},
		},
	}
	s.mockUsers.EXPECT().GetUser(s.requestContext, expectedUser.Id).Times(1).Return(nil, nil)
	s.mockUsers.EXPECT().Upsert(s.requestContext, expectedUser).Times(1).Return(nil)

	// Expect the user to have a group mapping to the None role.
	expectedGroup := &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "testuser@domain.tld",
		},
		RoleName: "None",
	}
	expectedAttributes := map[string][]string{
		"email": {"testuser@domain.tld"},
	}
	s.mockGroups.
		EXPECT().
		Walk(s.requestContext, fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*storage.Group{expectedGroup}, nil)

	// Expect the role to be fetched.
	expectedResolvedRole := roletest.NewResolvedRoleWithDenyAll(
		"None",
		utils.FromResourcesWithAccess(resources.AllResourcesViewPermissions()...))
	s.mockRoles.
		EXPECT().
		GetAndResolveRole(s.requestContext, "None").
		Times(1).
		Return(expectedResolvedRole, nil)

	userDescriptor := &permissions.UserDescriptor{
		UserID: "testuserid",
		Attributes: map[string][]string{
			"email": {"testuser@domain.tld"},
		},
	}
	roles, err := s.mapper.FromUserDescriptor(s.requestContext, userDescriptor)
	s.NoError(err, "mapping should have succeeded")
	s.ElementsMatch([]permissions.ResolvedRole{}, roles, "since only the None role was mapped, no role should be returned")
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
	s.mockUsers.EXPECT().GetUser(s.requestContext, expectedUser.Id).Times(1).Return(nil, nil)
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
	expectedResolvedRole1 := roletest.NewResolvedRoleWithDenyAll(
		"TeamAwesome",
		utils.FromResourcesWithAccess(resources.AllResourcesViewPermissions()...))
	expectedResolvedRole2 := roletest.NewResolvedRoleWithDenyAll(
		"TeamAwesome",
		utils.FromResourcesWithAccess(resources.AllResourcesModifyPermissions()...))
	s.mockRoles.
		EXPECT().
		GetAndResolveRole(s.requestContext, "TeamAwesome").
		Times(1).
		Return(expectedResolvedRole1, nil)
	s.mockRoles.
		EXPECT().
		GetAndResolveRole(s.requestContext, "TeamEvenAwesomer").
		Times(1).
		Return(expectedResolvedRole2, nil)

	// Call the mapper for a user.
	userDescriptor := &permissions.UserDescriptor{
		UserID: "coolguysid",
		Attributes: map[string][]string{
			"email": {"coolguy@yahoo"},
		},
	}
	roles, err := s.mapper.FromUserDescriptor(s.requestContext, userDescriptor)
	s.Require().NoError(err, "mapping should have succeeded")

	s.ElementsMatch([]permissions.ResolvedRole{expectedResolvedRole1, expectedResolvedRole2}, roles, "expected both roles to be present")
}

func (s *MapperTestSuite) TestMapperSuccessForMultipleRolesIncludingNone() {
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
	s.mockUsers.EXPECT().GetUser(s.requestContext, expectedUser.Id).Times(1).Return(nil, nil)
	s.mockUsers.EXPECT().Upsert(s.requestContext, expectedUser).Times(1).Return(nil)

	// Expect the user to have multiple group mappings for roles including None.
	expectedGroup := &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "coolguy@yahoo",
		},
		RoleName: "TeamAwesome",
	}
	expectedGroupNone := &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: fakeAuthProvider,
			Key:            "email",
			Value:          "coolguy@yahoo",
		},
		RoleName: "None",
	}
	expectedAttributes := map[string][]string{
		"email": {"coolguy@yahoo"},
	}
	s.mockGroups.
		EXPECT().
		Walk(s.requestContext, fakeAuthProvider, expectedAttributes).
		Times(1).
		Return([]*storage.Group{expectedGroup, expectedGroupNone}, nil)

	// Expect the roles to be fetched.
	expectedResolvedRole := roletest.NewResolvedRoleWithDenyAll(
		"TeamAwesome",
		utils.FromResourcesWithAccess(resources.AllResourcesViewPermissions()...))
	expectedResolvedNoneRole := roletest.NewResolvedRoleWithDenyAll(
		"None",
		utils.FromResourcesWithAccess(resources.AllResourcesViewPermissions()...))
	s.mockRoles.
		EXPECT().
		GetAndResolveRole(s.requestContext, "TeamAwesome").
		Times(1).
		Return(expectedResolvedRole, nil)
	s.mockRoles.
		EXPECT().
		GetAndResolveRole(s.requestContext, "None").
		Times(1).
		Return(expectedResolvedNoneRole, nil)

	userDescriptor := &permissions.UserDescriptor{
		UserID: "coolguysid",
		Attributes: map[string][]string{
			"email": {"coolguy@yahoo"},
		},
	}
	roles, err := s.mapper.FromUserDescriptor(s.requestContext, userDescriptor)
	s.NoError(err, "mapping should have succeeded")
	s.ElementsMatch([]permissions.ResolvedRole{expectedResolvedRole}, roles, "expected None role to be filtered out and the other one to be present")
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
	s.mockUsers.EXPECT().GetUser(s.requestContext, expectedUser.Id).Times(1).Return(nil, nil)
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
	expectedResolvedRole := roletest.NewResolvedRoleWithDenyAll(
		"TeamAwesome",
		utils.FromResourcesWithAccess(resources.AllResourcesViewPermissions()...))
	s.mockRoles.
		EXPECT().
		GetAndResolveRole(s.requestContext, "TeamAwesome").
		Times(1).
		Return(expectedResolvedRole, nil)

	// Call the mapper for a user.
	userDescriptor := &permissions.UserDescriptor{
		UserID: "coolguysid",
		Attributes: map[string][]string{
			"email": {"coolguy@yahoo"},
		},
	}
	roles, err := s.mapper.FromUserDescriptor(s.requestContext, userDescriptor)
	s.NoError(err, "mapping should have succeeded")
	s.ElementsMatch([]permissions.ResolvedRole{expectedResolvedRole}, roles, "since a single role was mapped, that role should be returned")
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
	s.mockUsers.EXPECT().GetUser(s.requestContext, expectedUser.Id).Times(1).Return(nil, nil)
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
	s.mockUsers.EXPECT().GetUser(s.requestContext, expectedUser.Id).Times(1).Return(nil, nil)
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
		GetAndResolveRole(s.requestContext, "TeamAwesome").
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
