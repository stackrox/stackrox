package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/suite"
)

func TestBoltClusters(t *testing.T) {
	suite.Run(t, new(BoltClusterTestSuite))
}

type BoltClusterTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltClusterTestSuite) SetupSuite() {
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

func (suite *BoltClusterTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltClusterTestSuite) TestClusters() {
	clusters := []*v1.Cluster{
		{
			Name:          "cluster1",
			MitigateImage: "test-dtr.example.com/mitigate",
		},
		{
			Name:          "cluster2",
			MitigateImage: "docker.io/stackrox/mitigate",
		},
	}

	// Test Add
	for _, b := range clusters {
		id, err := suite.AddCluster(b)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, b := range clusters {
		got, exists, err := suite.GetCluster(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Update
	for _, b := range clusters {
		b.MitigateImage = b.MitigateImage + "/mitigate"
	}

	for _, b := range clusters {
		suite.NoError(suite.UpdateCluster(b))
	}

	for _, b := range clusters {
		got, exists, err := suite.GetCluster(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Remove
	for _, b := range clusters {
		suite.NoError(suite.RemoveCluster(b.GetId()))
	}

	for _, b := range clusters {
		_, exists, err := suite.GetCluster(b.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
