//go:build sql_integration

package service

import (
	"context"
	"errors"
	"testing"

	deploymentDSMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	reportConfigurationDS "github.com/stackrox/rox/central/reports/config/datastore"
	datastoreMocks "github.com/stackrox/rox/central/resourcecollection/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestCollectionService(t *testing.T) {
	suite.Run(t, new(CollectionServiceTestSuite))
}

type CollectionServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	testDB            *pgtest.TestPostgres
	dataStore         *datastoreMocks.MockDataStore
	queryResolver     *datastoreMocks.MockQueryResolver
	deploymentDS      *deploymentDSMocks.MockDataStore
	resourceConfigDS  reportConfigurationDS.DataStore
	collectionService Service
}

func (suite *CollectionServiceTestSuite) SetupSuite() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.dataStore = datastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.queryResolver = datastoreMocks.NewMockQueryResolver(suite.mockCtrl)
	suite.deploymentDS = deploymentDSMocks.NewMockDataStore(suite.mockCtrl)

	suite.testDB = pgtest.ForT(suite.T())
	suite.resourceConfigDS = reportConfigurationDS.GetTestPostgresDataStore(suite.T(), suite.testDB.DB)
	suite.collectionService = New(suite.dataStore, suite.queryResolver, suite.deploymentDS, suite.resourceConfigDS)

	testutils.SetExampleVersion(suite.T())
}

func (suite *CollectionServiceTestSuite) TearDownSuite() {
	suite.testDB.Teardown(suite.T())
	suite.mockCtrl.Finish()
}

func (suite *CollectionServiceTestSuite) TestListCollectionSelectors() {
	selectorsResponse, err := suite.collectionService.ListCollectionSelectors(context.Background(), &v1.Empty{})
	suite.NoError(err)

	supportedLabelStrings := []string{
		search.Cluster.String(),
		search.ClusterLabel.String(),
		search.Namespace.String(),
		search.NamespaceLabel.String(),
		search.NamespaceAnnotation.String(),
		search.DeploymentName.String(),
		search.DeploymentLabel.String(),
		search.DeploymentAnnotation.String(),
	}

	suite.ElementsMatch(supportedLabelStrings, selectorsResponse.GetSelectors())
}

func (suite *CollectionServiceTestSuite) TestGetCollection() {
	request := &v1.GetCollectionRequest{
		Id: "a",
		Options: &v1.CollectionDeploymentMatchOptions{
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

	suite.dataStore.EXPECT().AddCollection(gomock.Any(), gomock.Any()).Times(1).Return("fake-id", nil)
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

	// test failure on datastore invocation
	mockID.EXPECT().UID().Return("uid").Times(1)
	mockID.EXPECT().FullName().Return("name").Times(1)
	mockID.EXPECT().FriendlyName().Return("name").Times(1)
	ctx = authn.ContextWithIdentity(allAccessCtx, mockID, suite.T())
	suite.dataStore.EXPECT().AddCollection(gomock.Any(), gomock.Any()).Times(1).Return("", errors.New("test error"))
	_, err = suite.collectionService.CreateCollection(ctx, request)
	suite.Error(err)
}

func (suite *CollectionServiceTestSuite) TestUpdateCollection() {
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

	// test failure on datastore invocation
	mockID.EXPECT().UID().Return("uid").Times(1)
	mockID.EXPECT().FullName().Return("name").Times(1)
	mockID.EXPECT().FriendlyName().Return("name").Times(1)
	ctx = authn.ContextWithIdentity(allAccessCtx, mockID, suite.T())
	suite.dataStore.EXPECT().UpdateCollection(ctx, gomock.Any()).Times(1).Return(errors.New("test error"))
	resp, err = suite.collectionService.UpdateCollection(ctx, request)
	suite.Error(err)
	suite.Nil(resp)
}

func (suite *CollectionServiceTestSuite) TestDeleteCollection() {
	allAccessCtx := sac.WithAllAccess(context.Background())

	// test error when ID is empty
	_, err := suite.collectionService.DeleteCollection(allAccessCtx, &v1.ResourceByID{})
	suite.Error(err)

	// test error when collectionId is in use by report config
	reportConfig := &storage.ReportConfiguration{
		Name:    "config0",
		ScopeId: "col0",
	}
	id, err := suite.resourceConfigDS.AddReportConfiguration(allAccessCtx, reportConfig)
	suite.NoError(err)
	idRequest := &v1.ResourceByID{Id: "col0"}
	_, err = suite.collectionService.DeleteCollection(allAccessCtx, idRequest)
	suite.Error(err)
	err = suite.resourceConfigDS.RemoveReportConfiguration(allAccessCtx, id)
	suite.NoError(err)

	// test successful deletion
	idRequest = &v1.ResourceByID{Id: "a"}
	suite.dataStore.EXPECT().DeleteCollection(allAccessCtx, idRequest.GetId()).Times(1).Return(nil)
	_, err = suite.collectionService.DeleteCollection(allAccessCtx, idRequest)
	suite.NoError(err)

	// test error when request fails
	suite.dataStore.EXPECT().DeleteCollection(allAccessCtx, idRequest.GetId()).Times(1).Return(errors.New("test error"))
	_, err = suite.collectionService.DeleteCollection(allAccessCtx, idRequest)
	suite.Error(err)
}

func (suite *CollectionServiceTestSuite) TestListCollections() {
	allAccessCtx := sac.WithAllAccess(context.Background())

	expectedResp := &v1.ListCollectionsResponse{
		Collections: []*storage.ResourceCollection{
			{
				Id: "test1",
			},
			{
				Id: "test2",
			},
		},
	}

	// test success
	suite.dataStore.EXPECT().SearchCollections(allAccessCtx, gomock.Any()).Times(1).Return(expectedResp.Collections, nil)
	resp, err := suite.collectionService.ListCollections(allAccessCtx, &v1.ListCollectionsRequest{})
	suite.NoError(err)
	suite.Equal(expectedResp, resp)

	// test failure
	suite.dataStore.EXPECT().SearchCollections(allAccessCtx, gomock.Any()).Times(1).Return(nil, errors.New("test error"))
	resp, err = suite.collectionService.ListCollections(allAccessCtx, &v1.ListCollectionsRequest{})
	suite.Error(err)
	suite.Nil(resp)
}

func (suite *CollectionServiceTestSuite) TestDryRunCollection() {
	allAccessCtx := sac.WithAllAccess(context.Background())

	expectedResp := &v1.DryRunCollectionResponse{
		Deployments: nil,
	}

	request := &v1.DryRunCollectionRequest{
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

	// test successful add request
	mockID := mockIdentity.NewMockIdentity(suite.mockCtrl)
	mockID.EXPECT().UID().Return("uid").Times(1)
	mockID.EXPECT().FullName().Return("name").Times(1)
	mockID.EXPECT().FriendlyName().Return("name").Times(1)
	ctx := authn.ContextWithIdentity(allAccessCtx, mockID, suite.T())

	suite.dataStore.EXPECT().DryRunAddCollection(ctx, gomock.Any()).Times(1).Return(nil)
	resp, err := suite.collectionService.DryRunCollection(ctx, request)
	suite.NoError(err)
	suite.Equal(expectedResp, resp)

	// test failure add request
	mockID.EXPECT().UID().Return("uid").Times(1)
	mockID.EXPECT().FullName().Return("name").Times(1)
	mockID.EXPECT().FriendlyName().Return("name").Times(1)
	ctx = authn.ContextWithIdentity(allAccessCtx, mockID, suite.T())

	suite.dataStore.EXPECT().DryRunAddCollection(ctx, gomock.Any()).Times(1).Return(errors.New("test error"))
	resp, err = suite.collectionService.DryRunCollection(ctx, request)
	suite.Error(err)
	suite.Nil(resp)

	// test successful update request
	mockID.EXPECT().UID().Return("uid").Times(1)
	mockID.EXPECT().FullName().Return("name").Times(1)
	mockID.EXPECT().FriendlyName().Return("name").Times(1)
	ctx = authn.ContextWithIdentity(allAccessCtx, mockID, suite.T())

	request.Id = "testId"

	suite.dataStore.EXPECT().DryRunUpdateCollection(ctx, gomock.Any()).Times(1).Return(nil)
	resp, err = suite.collectionService.DryRunCollection(ctx, request)
	suite.NoError(err)
	suite.Equal(expectedResp, resp)

	// test failure update request
	mockID.EXPECT().UID().Return("uid").Times(1)
	mockID.EXPECT().FullName().Return("name").Times(1)
	mockID.EXPECT().FriendlyName().Return("name").Times(1)
	ctx = authn.ContextWithIdentity(allAccessCtx, mockID, suite.T())
	suite.dataStore.EXPECT().DryRunUpdateCollection(ctx, gomock.Any()).Times(1).Return(errors.New("test error"))
	resp, err = suite.collectionService.DryRunCollection(ctx, request)
	suite.Error(err)
	suite.Nil(resp)

	// test deployment matching
	request.Options = &v1.CollectionDeploymentMatchOptions{
		WithMatches: true,
		FilterQuery: nil,
	}
	expectedResp = &v1.DryRunCollectionResponse{
		Deployments: []*storage.ListDeployment{
			{
				Id: "dep1",
			},
			{
				Id: "dep2",
			},
		},
	}
	mockID.EXPECT().UID().Return("uid").AnyTimes()
	mockID.EXPECT().FullName().Return("name").AnyTimes()
	mockID.EXPECT().FriendlyName().Return("name").AnyTimes()
	ctx = authn.ContextWithIdentity(allAccessCtx, mockID, suite.T())
	suite.dataStore.EXPECT().DryRunUpdateCollection(ctx, gomock.Any()).Times(1).Return(nil)
	suite.queryResolver.EXPECT().ResolveCollectionQuery(ctx, gomock.Any()).Times(1).Return(search.EmptyQuery(), nil)
	suite.deploymentDS.EXPECT().SearchListDeployments(ctx, gomock.Any()).Times(1).Return(expectedResp.Deployments, nil)
	resp, err = suite.collectionService.DryRunCollection(ctx, request)
	suite.NoError(err)
	suite.Equal(expectedResp, resp)

	// test failure to resolve query
	suite.dataStore.EXPECT().DryRunUpdateCollection(ctx, gomock.Any()).Times(1).Return(nil)
	suite.queryResolver.EXPECT().ResolveCollectionQuery(ctx, gomock.Any()).Times(1).Return(nil, errors.New("test error"))
	resp, err = suite.collectionService.DryRunCollection(ctx, request)
	suite.Error(err)
	suite.Nil(resp)
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
