package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	dsMocks "github.com/stackrox/rox/central/group/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestUserService(t *testing.T) {
	t.Parallel()
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
	update := &v1.GroupBatchUpdateRequest{
		PreviousGroups: []*storage.Group{
			{
				Props: &storage.GroupProperties{ // should be removed since the props are not in required
					AuthProviderId: "ap1",
					Key:            "k1",
					Value:          "v1",
				},
				RoleName: "r1",
			},
			{
				Props: &storage.GroupProperties{ // should be ignored since the props have the same role name in required
					AuthProviderId: "ap2",
					Key:            "k1",
					Value:          "v1",
				},
				RoleName: "r2",
			},
			{
				Props: &storage.GroupProperties{ // should get updated since the props have a new role in required
					AuthProviderId: "ap2",
					Key:            "k1",
					Value:          "v2",
				},
				RoleName: "r2",
			},
		},
		RequiredGroups: []*storage.Group{
			{
				Props: &storage.GroupProperties{ // repeat of the second group above
					AuthProviderId: "ap2",
					Key:            "k1",
					Value:          "v1",
				},
				RoleName: "r2",
			},
			{
				Props: &storage.GroupProperties{ // update to the third group above
					AuthProviderId: "ap2",
					Key:            "k1",
					Value:          "v2",
				},
				RoleName: "r3",
			},
			{
				Props: &storage.GroupProperties{ // newly added group since the props do not appear in previous.
					AuthProviderId: "ap2",
					Key:            "k2",
					Value:          "v1",
				},
				RoleName: "r3",
			},
		},
	}

	contextForMock := context.Background()
	suite.groupsMock.EXPECT().
		Mutate(contextForMock,
			[]*storage.Group{update.GetPreviousGroups()[0]},
			[]*storage.Group{update.GetRequiredGroups()[1]},
			[]*storage.Group{update.GetRequiredGroups()[2]}).
		Return(nil)

	_, err := suite.ser.BatchUpdate(contextForMock, update)
	suite.NoError(err, "request should not fail with valid user data")
}
