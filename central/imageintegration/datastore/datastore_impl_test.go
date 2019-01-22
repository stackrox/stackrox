package datastore

import (
	"os"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestImageIntegrationDataStore(t *testing.T) {
	suite.Run(t, new(ImageIntegrationDataStoreTestSuite))
}

type ImageIntegrationDataStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store     store.Store
	datastore DataStore
}

func (suite *ImageIntegrationDataStoreTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp(testutils.DBFileName(suite.Suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = store.New(db)
	suite.datastore = New(suite.store)
}

func (suite *ImageIntegrationDataStoreTestSuite) TearDownTest() {
	suite.db.Close()
	os.Remove(suite.db.Path())
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
		id, err := suite.datastore.AddImageIntegration(r)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	actualIntegrations, err := suite.datastore.GetImageIntegrations(&v1.GetImageIntegrationsRequest{})
	suite.NoError(err)
	suite.ElementsMatch(integrations, actualIntegrations)
}

func testIntegrations(t *testing.T, insertStorage store.Store, retrievalStorage DataStore) {
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
		got, exists, err := retrievalStorage.GetImageIntegration(r.GetId())
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
		got, exists, err := retrievalStorage.GetImageIntegration(r.GetId())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, r)
	}

	// Test Remove
	for _, r := range integrations {
		assert.NoError(t, insertStorage.RemoveImageIntegration(r.GetId()))
	}

	for _, r := range integrations {
		_, exists, err := retrievalStorage.GetImageIntegration(r.GetId())
		assert.NoError(t, err)
		assert.False(t, exists)
	}
}
