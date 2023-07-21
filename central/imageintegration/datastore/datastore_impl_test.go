//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/imageintegration/index"
	indexMocks "github.com/stackrox/rox/central/imageintegration/index/mocks"
	"github.com/stackrox/rox/central/imageintegration/search"
	searchMocks "github.com/stackrox/rox/central/imageintegration/search/mocks"
	"github.com/stackrox/rox/central/imageintegration/store"
	postgresStore "github.com/stackrox/rox/central/imageintegration/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	clusterID = "12841143-0b39-47be-8eca-5bca52b4a288"
)

func TestImageIntegrationDataStore(t *testing.T) {
	suite.Run(t, new(ImageIntegrationDataStoreTestSuite))
}

type ImageIntegrationDataStoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	mockIndexer  *indexMocks.MockIndexer
	mockSearcher *searchMocks.MockSearcher

	datastore DataStore

	testDB  *pgtest.TestPostgres
	store   store.Store
	indexer index.Indexer
}

func (suite *ImageIntegrationDataStoreTestSuite) SetupTest() {
	suite.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockIndexer = indexMocks.NewMockIndexer(suite.mockCtrl)
	suite.mockSearcher = searchMocks.NewMockSearcher(suite.mockCtrl)

	suite.testDB = pgtest.ForT(suite.T())
	suite.NotNil(suite.testDB)

	suite.store = postgresStore.New(suite.testDB.DB)
	suite.indexer = postgresStore.NewIndexer(suite.testDB.DB)

	// test formattedSearcher
	suite.datastore = NewForTestOnly(suite.store, suite.mockSearcher)
}

func (suite *ImageIntegrationDataStoreTestSuite) TearDownTest() {
	suite.testDB.Teardown(suite.T())
}

func (suite *ImageIntegrationDataStoreTestSuite) TestIntegrationsPersistence() {
	testIntegrations(suite.T(), suite.store, suite.datastore)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestIntegrations() {
	testIntegrations(suite.T(), suite.store, suite.datastore)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestIntegrationsFiltering() {
	integrations := []*storage.ImageIntegration{
		{
			Id:   uuid.NewV4().String(),
			Name: "registry1",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint1",
				},
			},
		},
		{
			Id:   uuid.NewV4().String(),
			Name: "registry2",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint2",
				},
			},
		},
	}
	// Test Add
	for _, r := range integrations {
		id, err := suite.datastore.AddImageIntegration(suite.hasWriteCtx, r)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	actualIntegrations, err := suite.datastore.GetImageIntegrations(suite.hasWriteCtx, &v1.GetImageIntegrationsRequest{})
	suite.NoError(err)
	suite.ElementsMatch(integrations, actualIntegrations)
}

func testIntegrations(t *testing.T, insertStorage store.Store, retrievalStorage DataStore) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Integration)))
	integrations := []*storage.ImageIntegration{
		{
			Id:   uuid.NewV4().String(),
			Name: "registry1",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint1",
				},
			},
		},
		{
			Id:   uuid.NewV4().String(),
			Name: "registry2",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint2",
				},
			},
		},
	}

	// Test Add
	for _, r := range integrations {
		err := insertStorage.Upsert(ctx, r)
		assert.NoError(t, err)
	}
	for _, r := range integrations {
		got, exists, err := retrievalStorage.GetImageIntegration(ctx, r.GetId())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, r)
	}

	// Test Update
	for _, r := range integrations {
		r.Name += "/api"
	}

	for _, r := range integrations {
		assert.NoError(t, insertStorage.Upsert(ctx, r))
	}

	for _, r := range integrations {
		got, exists, err := retrievalStorage.GetImageIntegration(ctx, r.GetId())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, r)
	}

	// Test Remove
	for _, r := range integrations {
		assert.NoError(t, insertStorage.Delete(ctx, r.GetId()))
	}

	for _, r := range integrations {
		_, exists, err := retrievalStorage.GetImageIntegration(ctx, r.GetId())
		assert.NoError(t, err)
		assert.False(t, exists)
	}
}

func getIntegration(name string) *storage.ImageIntegration {
	return &storage.ImageIntegration{
		Id:   uuid.NewV4().String(),
		Name: name,
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "https://endpoint1",
			},
		},
	}
}

func (suite *ImageIntegrationDataStoreTestSuite) storeIntegration(name string) *storage.ImageIntegration {
	integration := getIntegration(name)
	err := suite.store.Upsert(suite.hasWriteCtx, integration)
	suite.NoError(err)
	return integration
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesGet() {
	group, exists, err := suite.datastore.GetImageIntegration(suite.hasNoneCtx, "Some ID")
	suite.NoError(err, "expected no error, should return nil without access")
	suite.False(exists, "expected exists to be false as access was denied and bools can't be nil")
	suite.Nil(group, "expected return value to be nil")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsGet() {
	integration := suite.storeIntegration("Joseph Rules")

	gotInt, exists, err := suite.datastore.GetImageIntegration(suite.hasReadCtx, integration.GetId())
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.Equal(integration, gotInt)
	suite.True(exists)

	gotInt, exists, err = suite.datastore.GetImageIntegration(suite.hasWriteCtx, integration.GetId())
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.Equal(integration, gotInt)
	suite.True(exists)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesGetBatch() {
	integrations, err := suite.datastore.GetImageIntegrations(suite.hasNoneCtx, &v1.GetImageIntegrationsRequest{})
	suite.NoError(err, "expected no error, should return nil without access")
	suite.Nil(integrations, "expected return value to be nil")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsGetBatch() {
	integration := suite.storeIntegration("Some Integration")
	integrationList := []*storage.ImageIntegration{integration}

	getRequest := &v1.GetImageIntegrationsRequest{Name: integration.GetName(), Cluster: integration.GetClusterId()}

	gotImages, err := suite.datastore.GetImageIntegrations(suite.hasReadCtx, getRequest)
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.ElementsMatch(integrationList, gotImages)

	gotImages, err = suite.datastore.GetImageIntegrations(suite.hasWriteCtx, getRequest)
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.ElementsMatch(integrationList, gotImages)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesAdd() {
	integrationOne := getIntegration("some kinda name")
	id, err := suite.datastore.AddImageIntegration(suite.hasNoneCtx, integrationOne)
	suite.Error(err, "expected an error trying to write without permissions")
	suite.Empty(id)

	integrationTwo := getIntegration("Get named, you")
	id, err = suite.datastore.AddImageIntegration(suite.hasReadCtx, integrationTwo)
	suite.Error(err, "expected an error trying to write without permissions")
	suite.Empty(id)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsAdd() {
	id, err := suite.datastore.AddImageIntegration(suite.hasWriteCtx, getIntegration("namenamenamename"))
	suite.NoError(err, "expected no error trying to write with permissions")
	suite.NotEmpty(id)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesUpdate() {
	integration := suite.storeIntegration("name")

	err := suite.datastore.UpdateImageIntegration(suite.hasNoneCtx, integration)
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.UpdateImageIntegration(suite.hasReadCtx, integration)
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsUpdate() {
	integration := suite.storeIntegration("joseph is the best")

	err := suite.datastore.UpdateImageIntegration(suite.hasWriteCtx, integration)
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesRemove() {
	err := suite.datastore.RemoveImageIntegration(suite.hasNoneCtx, "blerk")
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.RemoveImageIntegration(suite.hasReadCtx, "hkddsfk")
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsRemove() {
	integration := suite.storeIntegration("jdgbfdkjh")

	err := suite.datastore.RemoveImageIntegration(suite.hasWriteCtx, integration.GetId())
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestSearch() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	suite.mockSearcher.EXPECT().Search(ctx, nil).Return(nil, nil)
	_, err := suite.datastore.Search(ctx, nil)
	suite.NoError(err)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestIndexing() {
	ii := &storage.ImageIntegration{
		Id:        uuid.NewV4().String(),
		ClusterId: clusterID,
		Name:      "imageIntegration1",
	}

	suite.NoError(suite.store.Upsert(sac.WithAllAccess(context.Background()), ii))

	q := pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, clusterID).ProtoQuery()
	results, err := suite.indexer.Search(suite.hasWriteCtx, q)
	suite.NoError(err)
	suite.Len(results, 1)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestDataStoreSearch() {
	ii := &storage.ImageIntegration{
		Id:        "id1",
		ClusterId: clusterID,
		Name:      "imageIntegration1",
	}

	// Create a new datastore since the one in suite uses mocks
	ds := New(suite.store, search.New(suite.store, suite.indexer))

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	_, err := ds.AddImageIntegration(ctx, ii)
	suite.NoError(err)

	q := pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, clusterID).ProtoQuery()
	results, err := ds.Search(ctx, q)
	suite.NoError(err)
	suite.Len(results, 1)
}
