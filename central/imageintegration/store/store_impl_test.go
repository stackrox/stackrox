package store

import (
	"os"
	"strings"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestImageIntegrationStore(t *testing.T) {
	suite.Run(t, new(ImageIntegrationStoreTestSuite))
}

type ImageIntegrationStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *ImageIntegrationStoreTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp("ImageIntegrationStoreTestSuite.db")
	if err != nil {
		suite.FailNow("failure: "+suite.T().Name(), err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *ImageIntegrationStoreTestSuite) TearDownTest() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *ImageIntegrationStoreTestSuite) TestIntegrations() {
	integration := []*v1.ImageIntegration{
		{
			Name: "registry1",
			IntegrationConfig: &v1.ImageIntegration_Docker{
				Docker: &v1.DockerConfig{
					Endpoint: "https://endpoint1",
				},
			},
		},
		{
			Name: "registry2",
			IntegrationConfig: &v1.ImageIntegration_Docker{
				Docker: &v1.DockerConfig{
					Endpoint: "https://endpoint2",
				},
			},
		},
	}

	// Test Add
	for _, r := range integration {
		id, err := suite.store.AddImageIntegration(r)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, r := range integration {
		got, exists, err := suite.store.GetImageIntegration(r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Update
	for _, r := range integration {
		r.Name += "-ext"
	}
	for _, r := range integration {
		suite.NoError(suite.store.UpdateImageIntegration(r))
	}
	for _, r := range integration {
		r.Name = strings.TrimSuffix(r.Name, "-ext")
	}
	for _, r := range integration {
		suite.NoError(suite.store.UpdateImageIntegration(r))
	}

	for _, r := range integration {
		got, exists, err := suite.store.GetImageIntegration(r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Remove
	for _, r := range integration {
		suite.NoError(suite.store.RemoveImageIntegration(r.GetId()))
	}

	for _, r := range integration {
		_, exists, err := suite.store.GetImageIntegration(r.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
