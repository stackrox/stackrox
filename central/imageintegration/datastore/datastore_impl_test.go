package datastore

import (
	"context"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestImageIntegrationDataStore(t *testing.T) {
	suite.Run(t, new(ImageIntegrationDataStoreTestSuite))
}

type ImageIntegrationDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	db *bolt.DB

	store     store.Store
	datastore DataStore
}

func (suite *ImageIntegrationDataStoreTestSuite) SetupTest() {
	suite.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.ImageIntegration)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ImageIntegration)))

	db, err := bolthelper.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = store.New(db)
	suite.datastore = New(suite.store)
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
	// Remove the default integrations
	for _, i := range store.DefaultImageIntegrations {
		suite.NoError(suite.datastore.RemoveImageIntegration(suite.hasWriteCtx, i.GetId()))
	}

	integrations := []*storage.ImageIntegration{
		{
			Name: "registry1",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint1",
				},
			},
		},
		{
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
			Name: "registry1",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint1",
				},
			},
		},
		{
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
		id, err := insertStorage.AddImageIntegration(r)
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
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
		assert.NoError(t, insertStorage.UpdateImageIntegration(r))
	}

	for _, r := range integrations {
		got, exists, err := retrievalStorage.GetImageIntegration(ctx, r.GetId())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, r)
	}

	// Test Remove
	for _, r := range integrations {
		assert.NoError(t, insertStorage.RemoveImageIntegration(r.GetId()))
	}

	for _, r := range integrations {
		_, exists, err := retrievalStorage.GetImageIntegration(ctx, r.GetId())
		assert.NoError(t, err)
		assert.False(t, exists)
	}
}

func getIntegration(name string) *storage.ImageIntegration {
	return &storage.ImageIntegration{
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
	id, err := suite.store.AddImageIntegration(integration)
	suite.NoError(err)
	suite.NotEmpty(id)
	return integration
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesGet() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	group, exists, err := suite.datastore.GetImageIntegration(suite.hasNoneCtx, "Some ID")
	suite.NoError(err, "expected no error, should return nil without access")
	suite.False(exists, "expected exists to be false as access was denied and bools can't be nil")
	suite.Nil(group, "expected return value to be nil")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsGet() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
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
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	integrations, err := suite.datastore.GetImageIntegrations(suite.hasNoneCtx, &v1.GetImageIntegrationsRequest{})
	suite.NoError(err, "expected no error, should return nil without access")
	suite.Nil(integrations, "expected return value to be nil")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsGetBatch() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
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
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
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
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	id, err := suite.datastore.AddImageIntegration(suite.hasWriteCtx, getIntegration("namenamenamename"))
	suite.NoError(err, "expected no error trying to write with permissions")
	suite.NotEmpty(id)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesUpdate() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	integration := suite.storeIntegration("name")

	err := suite.datastore.UpdateImageIntegration(suite.hasNoneCtx, integration)
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.UpdateImageIntegration(suite.hasReadCtx, integration)
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsUpdate() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	integration := suite.storeIntegration("joseph is the best")

	err := suite.datastore.UpdateImageIntegration(suite.hasWriteCtx, integration)
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesRemove() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	err := suite.datastore.RemoveImageIntegration(suite.hasNoneCtx, "blerk")
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.RemoveImageIntegration(suite.hasReadCtx, "hkddsfk")
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsRemove() {
	if !features.ScopedAccessControl.Enabled() {
		suite.T().Skip()
	}
	integration := suite.storeIntegration("jdgbfdkjh")

	err := suite.datastore.RemoveImageIntegration(suite.hasWriteCtx, integration.GetId())
	suite.NoError(err, "expected no error trying to write with permissions")
}
