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

func (suite *CollectionServiceTestSuite) TestGetCollectionCount() {
	if !features.ObjectCollections.Enabled() {
		suite.T().Skip("skipping because env var is not set")
	}

	allAccessCtx := sac.WithAllAccess(context.Background())
	request := &v1.GetCollectionCountRequest{
		Query: &v1.RawQuery{},
	}

	parsedQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	suite.NoError(err)

	suite.dataStore.EXPECT().Count(allAccessCtx, parsedQuery).Times(1).Return(10, nil)
	resp, err := suite.collectionService.GetCollectionCount(allAccessCtx, request)
	suite.NoError(err)
	suite.NotNil(resp)
	suite.Equal(int32(10), resp.GetCount())

	// test error
	suite.dataStore.EXPECT().Count(allAccessCtx, parsedQuery).Times(1).Return(0, errors.New("test error"))
	resp, err = suite.collectionService.GetCollectionCount(allAccessCtx, request)
	suite.Error(err)
	suite.Nil(resp)
}

func (suite *CollectionServiceTestSuite) TestCreateCollection() {
	if !features.ObjectCollections.Enabled() {
		suite.T().Skip("skipping because env var is not set")
	}
	allAccessCtx := sac.WithAllAccess(context.Background())

	// test error when collection name is empty
	request := &v1.CreateCollectionRequest{
		Name: "",
	}
	resp, err := suite.collectionService.CreateCollection(allAccessCtx, request)
	suite.NotNil(err)
	suite.Nil(resp)

	// test error on context without identity
	request = &v1.CreateCollectionRequest{
		Name: "b",
	}
	resp, err = suite.collectionService.CreateCollection(allAccessCtx, request)
	suite.NotNil(err)
	suite.Nil(resp)

	// test error on empty/nil resource selectors
	request = &v1.CreateCollectionRequest{
		Name: "c",
	}
	mockID := mockIdentity.NewMockIdentity(suite.mockCtrl)
	mockID.EXPECT().UID().Return("uid").Times(1)
	mockID.EXPECT().FullName().Return("name").Times(1)
	mockID.EXPECT().FriendlyName().Return("name").Times(1)
	ctx := authn.ContextWithIdentity(allAccessCtx, mockID, suite.T())
	resp, err = suite.collectionService.CreateCollection(ctx, request)
	suite.NotNil(err)
	suite.Nil(resp)

	// test successful collection creation
	request = &v1.CreateCollectionRequest{
		Name:        "d",
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

	mockID.EXPECT().UID().Return("uid").Times(1)
	mockID.EXPECT().FullName().Return("name").Times(1)
	mockID.EXPECT().FriendlyName().Return("name").Times(1)
	ctx = authn.ContextWithIdentity(allAccessCtx, mockID, suite.T())

	suite.dataStore.EXPECT().AddCollection(gomock.Any(), gomock.Any()).Times(1).Return(nil)
	resp, err = suite.collectionService.CreateCollection(ctx, request)
	suite.NoError(err)
	suite.NotNil(resp.GetCollection())
	suite.Equal(request.Name, resp.GetCollection().GetName())
	suite.Equal(request.GetDescription(), resp.GetCollection().GetDescription())
	suite.Equal(request.GetResourceSelectors(), resp.GetCollection().GetResourceSelectors())
	suite.NotNil(resp.GetCollection().GetEmbeddedCollections())
	suite.Equal(request.GetEmbeddedCollectionIds(), suite.embeddedCollectionsToIds(resp.GetCollection().GetEmbeddedCollections()))
	suite.NotNil(resp.GetCollection().GetCreatedBy())
	suite.NotNil(resp.GetCollection().GetUpdatedBy())
	suite.NotNil(resp.GetCollection().GetCreatedAt())
	suite.NotNil(resp.GetCollection().GetLastUpdated())
}

func (suite *CollectionServiceTestSuite) TestUpdateCollection() {
	if !features.ObjectCollections.Enabled() {
		suite.T().Skip("skipping because env var is not set")
	}
	allAccessCtx := sac.WithAllAccess(context.Background())

	// test error when collection Id is empty
	request := &v1.UpdateCollectionRequest{
		Id: "",
	}
	resp, err := suite.collectionService.UpdateCollection(allAccessCtx, request)
	suite.NotNil(err)
	suite.Nil(resp)

	// test error when collection name is empty
	request = &v1.UpdateCollectionRequest{
		Id:   "id1",
		Name: "",
	}
	resp, err = suite.collectionService.UpdateCollection(allAccessCtx, request)
	suite.NotNil(err)
	suite.Nil(resp)

	// test error on context without identity
	request = &v1.UpdateCollectionRequest{
		Id:   "id2",
		Name: "b",
	}
	resp, err = suite.collectionService.UpdateCollection(allAccessCtx, request)
	suite.NotNil(err)
	suite.Nil(resp)

	// test error on empty/nil resource selectors
	request = &v1.UpdateCollectionRequest{
		Id:   "id3",
		Name: "c",
	}
	mockID := mockIdentity.NewMockIdentity(suite.mockCtrl)
	mockID.EXPECT().UID().Return("uid").Times(1)
	mockID.EXPECT().FullName().Return("name").Times(1)
	mockID.EXPECT().FriendlyName().Return("name").Times(1)
	ctx := authn.ContextWithIdentity(allAccessCtx, mockID, suite.T())
	resp, err = suite.collectionService.UpdateCollection(ctx, request)
	suite.NotNil(err)
	suite.Nil(resp)

	// test successful update
	request = &v1.UpdateCollectionRequest{
		Id:          "id4",
		Name:        "d",
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

	mockID.EXPECT().UID().Return("uid").Times(1)
	mockID.EXPECT().FullName().Return("name").Times(1)
	mockID.EXPECT().FriendlyName().Return("name").Times(1)
	ctx = authn.ContextWithIdentity(allAccessCtx, mockID, suite.T())

	suite.dataStore.EXPECT().UpdateCollection(ctx, gomock.Any()).Times(1).Return(nil)
	resp, err = suite.collectionService.UpdateCollection(ctx, request)
	suite.NoError(err)
	suite.NotNil(resp.GetCollection())
	suite.Equal(request.GetId(), resp.GetCollection().GetId())
	suite.Equal(request.Name, resp.GetCollection().GetName())
	suite.Equal(request.GetDescription(), resp.GetCollection().GetDescription())
	suite.Equal(request.GetResourceSelectors(), resp.GetCollection().GetResourceSelectors())
	suite.Equal(request.GetEmbeddedCollectionIds(), suite.embeddedCollectionsToIds(resp.GetCollection().GetEmbeddedCollections()))
	suite.Equal("uid", resp.GetCollection().GetUpdatedBy().GetId())
	suite.Equal("name", resp.GetCollection().GetUpdatedBy().GetName())
	suite.NotNil(resp.GetCollection().GetLastUpdated())
}

func (suite *CollectionServiceTestSuite) TestDeleteCollection() {
	if !features.ObjectCollections.Enabled() {
		suite.T().Skip("skipping because env var is not set")
	}
	allAccessCtx := sac.WithAllAccess(context.Background())

	// test error when ID is empty
	_, err := suite.collectionService.DeleteCollection(allAccessCtx, &v1.ResourceByID{})
	suite.Error(err)

	// test successful deletion
	idRequest := &v1.ResourceByID{Id: "a"}
	suite.dataStore.EXPECT().DeleteCollection(allAccessCtx, idRequest.GetId()).Times(1).Return(nil)
	_, err = suite.collectionService.DeleteCollection(allAccessCtx, idRequest)
	suite.NoError(err)
}

func (suite *CollectionServiceTestSuite) embeddedCollectionsToIds(embeddedCollections []*storage.ResourceCollection_EmbeddedResourceCollection) []string {
	if len(embeddedCollections) == 0 {
		return nil
	}
	ids := make([]string, 0, len(embeddedCollections))
	for _, c := range embeddedCollections {
		ids = append(ids, c.GetId())
	}
	return ids
}
