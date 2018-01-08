package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltRegistries(t *testing.T) {
	suite.Run(t, new(BoltRegistryTestSuite))
}

type BoltRegistryTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltRegistryTestSuite) SetupSuite() {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		suite.FailNow("Failed to get temporary directory", err.Error())
	}
	db, err := New(tmpDir)
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db
}

func (suite *BoltRegistryTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltRegistryTestSuite) TestDeployments() {
	registries := []*v1.Registry{
		{
			Name:     "registry1",
			Endpoint: "https://endpoint1",
		},
		{
			Name:     "registry2",
			Endpoint: "https://endpoint2",
		},
	}

	// Test Add
	for _, r := range registries {
		suite.NoError(suite.AddRegistry(r))
	}

	for _, r := range registries {
		got, exists, err := suite.GetRegistry(r.Name)
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Update
	for _, r := range registries {
		r.Endpoint += "/api"
	}

	for _, r := range registries {
		suite.NoError(suite.UpdateRegistry(r))
	}

	for _, r := range registries {
		got, exists, err := suite.GetRegistry(r.Name)
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Remove
	for _, r := range registries {
		suite.NoError(suite.RemoveRegistry(r.Name))
	}

	for _, r := range registries {
		_, exists, err := suite.GetRegistry(r.Name)
		suite.NoError(err)
		suite.False(exists)
	}
}
