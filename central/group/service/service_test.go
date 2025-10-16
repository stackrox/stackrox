package service

import (
	"context"
	"testing"

	dsMocks "github.com/stackrox/rox/central/group/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestUserService(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}

type UserServiceTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller

	groupsMock *dsMocks.MockDataStore
	ser        Service
}

func (suite *UserServiceTestSuite) SetupSuite() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.groupsMock = dsMocks.NewMockDataStore(suite.mockCtrl)
	suite.ser = New(suite.groupsMock)
}

func (suite *UserServiceTestSuite) TestBatchUpdate() {
	update := v1.GroupBatchUpdateRequest_builder{
		PreviousGroups: []*storage.Group{
			storage.Group_builder{
				Props: storage.GroupProperties_builder{ // should be removed since the props are not in required
					AuthProviderId: "ap1",
					Key:            "k1",
					Value:          "v1",
					Id:             "1",
				}.Build(),
				RoleName: "r1",
			}.Build(),
			storage.Group_builder{
				Props: storage.GroupProperties_builder{ // should be ignored since the props have the same role name in required
					AuthProviderId: "ap2",
					Key:            "k1",
					Value:          "v1",
					Id:             "2",
				}.Build(),
				RoleName: "r2",
			}.Build(),
			storage.Group_builder{
				Props: storage.GroupProperties_builder{ // should get updated since the props have a new role in required
					AuthProviderId: "ap2",
					Key:            "k1",
					Value:          "v2",
					Id:             "3",
				}.Build(),
				RoleName: "r2",
			}.Build(),
		},
		RequiredGroups: []*storage.Group{
			storage.Group_builder{
				Props: storage.GroupProperties_builder{ // repeat of the second group above
					AuthProviderId: "ap2",
					Key:            "k1",
					Value:          "v1",
					Id:             "2",
				}.Build(),
				RoleName: "r2",
			}.Build(),
			storage.Group_builder{
				Props: storage.GroupProperties_builder{ // update to the third group above
					AuthProviderId: "ap2",
					Key:            "k1",
					Value:          "v2",
					Id:             "3",
				}.Build(),
				RoleName: "r3",
			}.Build(),
			storage.Group_builder{
				Props: storage.GroupProperties_builder{ // newly added group since the props do not appear in previous.
					AuthProviderId: "ap2",
					Key:            "k2",
					Value:          "v1",
				}.Build(),
				RoleName: "r4",
			}.Build(),
			storage.Group_builder{
				Props: storage.GroupProperties_builder{ // repeat of the second group above
					AuthProviderId: "ap2",
					Key:            "k1",
					Value:          "v1",
				}.Build(),
				RoleName: "r2",
			}.Build(),
		},
	}.Build()

	contextForMock := context.Background()
	suite.groupsMock.EXPECT().
		Mutate(contextForMock,
			[]*storage.Group{update.GetPreviousGroups()[0]},
			[]*storage.Group{update.GetRequiredGroups()[1]},
			[]*storage.Group{update.GetRequiredGroups()[2]}, false).
		Return(nil)

	_, err := suite.ser.BatchUpdate(contextForMock, update)
	suite.NoError(err, "request should not fail with valid user data")
}

func (suite *UserServiceTestSuite) TestBatchUpdate_Dedupe_updated_group() {
	update := v1.GroupBatchUpdateRequest_builder{
		PreviousGroups: []*storage.Group{
			storage.Group_builder{
				Props: storage.GroupProperties_builder{
					AuthProviderId: "ap1",
					Key:            "k2",
					Value:          "v1",
					Id:             "1",
				}.Build(),
				RoleName: "r1",
			}.Build(),
		},
		RequiredGroups: []*storage.Group{
			storage.Group_builder{
				Props: storage.GroupProperties_builder{ // update of the first group in previous groups.
					AuthProviderId: "ap2",
					Key:            "k1",
					Value:          "v1",
					Id:             "1",
				}.Build(),
				RoleName: "r2",
			}.Build(),
			storage.Group_builder{
				Props: storage.GroupProperties_builder{ // dupe of the first group in required groups, should not be added.
					AuthProviderId: "ap2",
					Key:            "k1",
					Value:          "v1",
				}.Build(),
				RoleName: "r2",
			}.Build(),
		},
	}.Build()

	contextForMock := context.Background()
	suite.groupsMock.EXPECT().
		Mutate(contextForMock,
			gomock.Len(0),
			[]*storage.Group{update.GetRequiredGroups()[0]},
			gomock.Len(0), false).
		Return(nil)

	_, err := suite.ser.BatchUpdate(contextForMock, update)
	suite.NoError(err, "request should not fail with valid user data")
}
