package store

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
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

func (suite *ImageIntegrationStoreTestSuite) TeardownTest() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *ImageIntegrationStoreTestSuite) TestIntegrations() {
	integration := []*v1.ImageIntegration{
		{
			Name: "registry1",
			Config: map[string]string{
				"endpoint": "https://endpoint1",
			},
		},
		{
			Name: "registry2",
			Config: map[string]string{
				"endpoint": "https://endpoint2",
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
