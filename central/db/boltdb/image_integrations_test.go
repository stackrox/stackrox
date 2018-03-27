package boltdb

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltImageIntegration(t *testing.T) {
	suite.Run(t, new(BoltImageIntegrationTestSuite))
}

type BoltImageIntegrationTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltImageIntegrationTestSuite) SetupSuite() {
	db, err := boltFromTmpDir()
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db
}

func (suite *BoltImageIntegrationTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltImageIntegrationTestSuite) TestIntegrations() {
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
		id, err := suite.AddImageIntegration(r)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, r := range integration {
		got, exists, err := suite.GetImageIntegration(r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Update
	for _, r := range integration {
		r.Name += "-ext"
	}

	for _, r := range integration {
		suite.NoError(suite.UpdateImageIntegration(r))
	}

	for _, r := range integration {
		got, exists, err := suite.GetImageIntegration(r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Remove
	for _, r := range integration {
		suite.NoError(suite.RemoveImageIntegration(r.GetId()))
	}

	for _, r := range integration {
		_, exists, err := suite.GetImageIntegration(r.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
