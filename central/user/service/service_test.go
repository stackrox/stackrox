package service

import (
	"context"
	"testing"

	dsMocks "github.com/stackrox/rox/central/user/datastore/mocks"
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

	usersMock *dsMocks.MockDataStore
	ser       Service
}

func (suite *UserServiceTestSuite) SetupSuite() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.usersMock = dsMocks.NewMockDataStore(suite.mockCtrl)
	suite.ser = New(suite.usersMock)
}

func (suite *UserServiceTestSuite) TestGetUsersAttributes() {
	expectedContext := context.Background()
	users := []*storage.User{
		storage.User_builder{
			Id:             "user1",
			AuthProviderId: "ap1",
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
			Id:             "user2",
			AuthProviderId: "ap1",
			Attributes: []*storage.UserAttribute{
				storage.UserAttribute_builder{
					Key:   "name",
					Value: "user2",
				}.Build(),
			},
		}.Build(),
		storage.User_builder{
			Id:             "user3",
			AuthProviderId: "ap2",
			Attributes: []*storage.UserAttribute{
				storage.UserAttribute_builder{
					Key:   "name",
					Value: "user1",
				}.Build(),
			},
		}.Build(),
		storage.User_builder{
			Id:             "user4",
			AuthProviderId: "ap1",
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
	}

	expectedAttributes := []*v1.UserAttributeTuple{
		v1.UserAttributeTuple_builder{
			AuthProviderId: "ap1",
			Key:            "name",
			Value:          "user1",
		}.Build(),
		v1.UserAttributeTuple_builder{
			AuthProviderId: "ap1",
			Key:            "email",
			Value:          "user@derp.com",
		}.Build(),
		v1.UserAttributeTuple_builder{
			AuthProviderId: "ap1",
			Key:            "name",
			Value:          "user2",
		}.Build(),
		v1.UserAttributeTuple_builder{
			AuthProviderId: "ap2",
			Key:            "name",
			Value:          "user1",
		}.Build(),
	}

	suite.usersMock.EXPECT().GetAllUsers(expectedContext).Return(users, nil)

	resp, err := suite.ser.GetUsersAttributes(expectedContext, nil)
	suite.NoError(err, "request should not fail with valid user data")

	suite.Equal(len(expectedAttributes), len(resp.GetUsersAttributes()))
}
