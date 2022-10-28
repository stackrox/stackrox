package service

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	datastoreMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

func TestCollectionService(t *testing.T) {
	suite.Run(t, new(CollectionServiceTestSuite))
}

type CollectionServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	dataStore         *datastoreMocks.MockDataStore
	collectionService Service
}

func (suite *CollectionServiceTestSuite) SetupSuite() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.dataStore = datastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.T().Setenv(features.ObjectCollections.EnvVar(), "true")
	suite.collectionService = New(suite.dataStore)

	testbuildinfo.SetForTest(suite.T())
	testutils.SetExampleVersion(suite.T())
}

func (suite *CollectionServiceTestSuite) TearDownSuite() {
	suite.mockCtrl.Finish()
}

func (suite *CollectionServiceTestSuite) TestGetCollection() {
	if !features.ObjectCollections.Enabled() {
		suite.T().Skip("skipping because env var is not set")
	}

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

	expected := &v1.GetCollectionResponse{
		Collection:  collection,
		Deployments: nil,
	}

	result, err := suite.collectionService.GetCollection(context.Background(), request)
	suite.NoError(err)
	suite.Equal(expected, result)

	// collection not present
	suite.dataStore.EXPECT().Get(gomock.Any(), request.Id).Times(1).Return(nil, false, nil)

	result, err = suite.collectionService.GetCollection(context.Background(), request)
	suite.NotNil(err)
	suite.Nil(result)

	// error
	suite.dataStore.EXPECT().Get(gomock.Any(), request.Id).Times(1).Return(nil, false, errors.New("test error"))

	result, err = suite.collectionService.GetCollection(context.Background(), request)
	suite.NotNil(err)
	suite.Nil(result)
}

func (suite *CollectionServiceTestSuite) TestCreateCollection() {
	if !features.ObjectCollections.Enabled() {
		suite.T().Skip("skipping because env var is not set")
	}
	ctx := sac.WithAllAccess(context.Background())

	// test error when collection name is empty
	request := &v1.CreateCollectionRequest{
		Name: "",
	}
	resp, err := suite.collectionService.CreateCollection(ctx, request)
	suite.NotNil(err)
	suite.Nil(resp)

	// test error on context without identity
	request = &v1.CreateCollectionRequest{
		Name: "b",
	}
	resp, err = suite.collectionService.CreateCollection(ctx, request)
	suite.NotNil(err)
	suite.Nil(resp)

	// test successful collection creation
	request = &v1.CreateCollectionRequest{
		Name:        "c",
		Description: "description",
		ResourceSelectors: []*storage.ResourceSelector{
			{
				Rules: []*storage.SelectorRule{
					{
						FieldName: search.DeploymentName.String(),
						Operator:  storage.BooleanOperator_OR,
						Values: []*storage.RuleValue{
							{
								Value: "deployment",
							},
						},
					},
				},
			},
		},
		EmbeddedCollectionIds: []string{"id1", "id2"},
	}

	mockIdentity := mockIdentity.NewMockIdentity(suite.mockCtrl)
	mockIdentity.EXPECT().UID().Return("uid").Times(1)
	mockIdentity.EXPECT().FullName().Return("name").Times(1)
	mockIdentity.EXPECT().FriendlyName().Return("name").Times(1)
	ctx = authn.ContextWithIdentity(ctx, mockIdentity, suite.T())

	suite.dataStore.EXPECT().AddCollection(gomock.Any(), gomock.Any()).Times(1).Return(nil)
	resp, err = suite.collectionService.CreateCollection(ctx, request)
	suite.NoError(err)
	suite.NotNil(resp.GetCollection())
	suite.Equal(request.Name, resp.GetCollection().GetName())
	suite.Equal(request.GetDescription(), resp.GetCollection().GetDescription())
	suite.Equal(request.GetResourceSelectors(), resp.GetCollection().GetResourceSelectors())
	suite.NotNil(resp.GetCollection().GetEmbeddedCollections())
	suite.Equal(request.GetEmbeddedCollectionIds(), suite.embeddedCollectionToIds(resp.GetCollection().GetEmbeddedCollections()))
}

func (suite *CollectionServiceTestSuite) embeddedCollectionToIds(embeddedCollections []*storage.ResourceCollection_EmbeddedResourceCollection) []string {
	if len(embeddedCollections) == 0 {
		return nil
	}
	ids := make([]string, 0, len(embeddedCollections))
	for _, c := range embeddedCollections {
		ids = append(ids, c.GetId())
	}
	return ids
}
