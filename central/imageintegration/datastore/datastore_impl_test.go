package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	indexMocks "github.com/stackrox/rox/central/imageintegration/index/mocks"
	"github.com/stackrox/rox/central/imageintegration/store"
	boltStore "github.com/stackrox/rox/central/imageintegration/store/bolt"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestImageIntegrationDataStore(t *testing.T) {
	suite.Run(t, new(ImageIntegrationDataStoreTestSuite))
}

type ImageIntegrationDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	hasReadIntegrationsCtx  context.Context
	hasWriteIntegrationsCtx context.Context

	db *bolt.DB

	store     store.Store
	datastore DataStore

	indexer  *indexMocks.MockIndexer
	mockCtrl *gomock.Controller
}

func (suite *ImageIntegrationDataStoreTestSuite) SetupTest() {
	suite.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.ImageIntegration)))
	suite.hasReadIntegrationsCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ImageIntegration)))
	suite.hasWriteIntegrationsCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))

	db, err := bolthelper.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = boltStore.New(db)

	// test searcher
	suite.datastore = NewForTestOnly(suite.T(), suite.store, suite.indexer, nil)
}

func (suite *ImageIntegrationDataStoreTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
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
		sac.ResourceScopeKeys(resources.ImageIntegration)))
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
	err := suite.store.Upsert(suite.hasReadCtx, integration)
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

	gotInt, exists, err = suite.datastore.GetImageIntegration(suite.hasReadIntegrationsCtx, integration.GetId())
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.Equal(integration, gotInt)
	suite.True(exists)

	gotInt, exists, err = suite.datastore.GetImageIntegration(suite.hasWriteCtx, integration.GetId())
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.Equal(integration, gotInt)
	suite.True(exists)

	gotInt, exists, err = suite.datastore.GetImageIntegration(suite.hasWriteIntegrationsCtx, integration.GetId())
	suite.NoError(err, "expected no error trying to read with Integration permissions")
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

	gotImages, err = suite.datastore.GetImageIntegrations(suite.hasReadIntegrationsCtx, getRequest)
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.ElementsMatch(integrationList, gotImages)

	gotImages, err = suite.datastore.GetImageIntegrations(suite.hasWriteCtx, getRequest)
	suite.NoError(err, "expected no error trying to read with permissions")
	suite.ElementsMatch(integrationList, gotImages)

	gotImages, err = suite.datastore.GetImageIntegrations(suite.hasWriteIntegrationsCtx, getRequest)
	suite.NoError(err, "expected no error trying to read with Integration permissions")
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

	id, err = suite.datastore.AddImageIntegration(suite.hasReadIntegrationsCtx, integrationTwo)
	suite.Error(err, "expected an error trying to write without permissions")
	suite.Empty(id)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsAdd() {
	id, err := suite.datastore.AddImageIntegration(suite.hasWriteCtx, getIntegration("namenamenamename"))
	suite.NoError(err, "expected no error trying to write with permissions")
	suite.NotEmpty(id)

	id, err = suite.datastore.AddImageIntegration(suite.hasWriteIntegrationsCtx, getIntegration("namenamenamename2"))
	suite.NoError(err, "expected no error trying to write with permissions")
	suite.NotEmpty(id)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesUpdate() {
	integration := suite.storeIntegration("name")

	err := suite.datastore.UpdateImageIntegration(suite.hasNoneCtx, integration)
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.UpdateImageIntegration(suite.hasReadCtx, integration)
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.UpdateImageIntegration(suite.hasReadIntegrationsCtx, integration)
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsUpdate() {
	integration := suite.storeIntegration("joseph is the best")

	err := suite.datastore.UpdateImageIntegration(suite.hasWriteCtx, integration)
	suite.NoError(err, "expected no error trying to write with permissions")

	integration = suite.storeIntegration("joseph is the best again")

	err = suite.datastore.UpdateImageIntegration(suite.hasWriteIntegrationsCtx, integration)
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesRemove() {
	err := suite.datastore.RemoveImageIntegration(suite.hasNoneCtx, "blerk")
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.RemoveImageIntegration(suite.hasReadCtx, "hkddsfk")
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.RemoveImageIntegration(suite.hasReadIntegrationsCtx, "hkddsfk2")
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsRemove() {
	integration := suite.storeIntegration("jdgbfdkjh")

	err := suite.datastore.RemoveImageIntegration(suite.hasWriteCtx, integration.GetId())
	suite.NoError(err, "expected no error trying to write with permissions")

	integration = suite.storeIntegration("jdgbfdkjh2")

	err = suite.datastore.RemoveImageIntegration(suite.hasWriteIntegrationsCtx, integration.GetId())
	suite.NoError(err, "expected no error trying to write with permissions")
}
