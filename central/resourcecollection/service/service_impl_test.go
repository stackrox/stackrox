package service

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	configDatastoreMocks "github.com/stackrox/rox/central/config/datastore/mocks"
	datastoreMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

func TestCollectionService(t *testing.T) {
	suite.Run(t, new(CollectionServiceTestSuite))
}

type CollectionServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller
	ei       *envisolator.EnvIsolator

	dataStore          *datastoreMocks.MockDataStore
	sysConfigDatastore *configDatastoreMocks.MockDataStore
}

var _ suite.TearDownTestSuite = (*CollectionServiceTestSuite)(nil)

func (suite *CollectionServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.dataStore = datastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.sysConfigDatastore = configDatastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.ei = envisolator.NewEnvIsolator(suite.T())

	suite.ei.Setenv("ROX_IMAGE_FLAVOR", "rhacs")
	testbuildinfo.SetForTest(suite.T())
	testutils.SetExampleVersion(suite.T())
}

func (suite *CollectionServiceTestSuite) TearDownTest() {
	suite.ei.RestoreAll()
	suite.mockCtrl.Finish()
}

func (suite *CollectionServiceTestSuite) TestGetCollection() {
	request := &v1.GetCollectionRequest{
		Id: "a",
		Options: &v1.GetCollectionRequest_Options{
			WithMatches: false,
		},
	}
	collection := &storage.ResourceCollection{
		Id: "a",
	}
	// successful get
	suite.dataStore.EXPECT().Get(gomock.Any(), request.Id).Times(1).Return(collection, true, nil)
	collectionService := New(suite.dataStore)

	expected := &v1.GetCollectionResponse{
		Collection:  collection,
		Deployments: nil,
	}

	result, err := collectionService.GetCollection(context.Background(), request)
	suite.NoError(err)
	suite.Equal(expected, result)

	// collection not present
	suite.dataStore.EXPECT().Get(gomock.Any(), request.Id).Times(1).Return(nil, false, nil)
	collectionService = New(suite.dataStore)

	result, err = collectionService.GetCollection(context.Background(), request)
	suite.NotNil(err)
	suite.Nil(result)

	// error
	suite.dataStore.EXPECT().Get(gomock.Any(), request.Id).Times(1).Return(nil, false, errors.New("test error"))
	collectionService = New(suite.dataStore)

	result, err = collectionService.GetCollection(context.Background(), request)
	suite.NotNil(err)
	suite.Nil(result)
}
