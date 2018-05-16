package boltdb

import (
	"os"
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	ptypes "github.com/gogo/protobuf/types"
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
	db, err := boltFromTmpDir()
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
	checkin1 := time.Now()
	checkin2 := time.Now().Add(-1 * time.Hour)
	ts1, err := ptypes.TimestampProto(checkin1)
	suite.NoError(err)
	ts2, err := ptypes.TimestampProto(checkin2)
	suite.NoError(err)

	clusters := []*v1.Cluster{
		{
			Name:         "cluster1",
			PreventImage: "test-dtr.example.com/prevent",
			LastContact:  ts1,
		},
		{
			Name:         "cluster2",
			PreventImage: "docker.io/stackrox/prevent",
			LastContact:  ts2,
		},
	}

	// Test Add
	for _, b := range clusters {
		id, err := suite.AddCluster(b)
		suite.NoError(err)
		suite.NotEmpty(id)

		// Add the timestamp in the second list.
		t, err := ptypes.TimestampFromProto(b.GetLastContact())
		suite.NoError(err)
		err = suite.UpdateClusterContactTime(b.GetId(), t)
		suite.NoError(err)
	}

	for _, b := range clusters {
		got, exists, err := suite.GetCluster(b.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, b)
	}

	// Test Update
	for _, b := range clusters {
		b.PreventImage = b.PreventImage + "/prevent"
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

	// Test Count
	count, err := suite.CountClusters()
	suite.NoError(err)
	suite.Equal(len(clusters), count)

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
