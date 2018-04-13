package boltdb

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/suite"
)

func TestBoltDeployments(t *testing.T) {
	suite.Run(t, new(BoltDeploymentTestSuite))
}

type BoltDeploymentTestSuite struct {
	suite.Suite
	*BoltDB
}

func (suite *BoltDeploymentTestSuite) SetupSuite() {
	db, err := boltFromTmpDir()
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db
}

func (suite *BoltDeploymentTestSuite) TeardownSuite() {
	suite.Close()
	os.Remove(suite.Path())
}

func (suite *BoltDeploymentTestSuite) TestDeployments() {
	deployments := []*v1.Deployment{
		{
			Id:        "fooID",
			Name:      "foo",
			Version:   "100",
			Type:      "Replicated",
			UpdatedAt: ptypes.TimestampNow(),
		},
		{
			Id:        "barID",
			Name:      "bar",
			Version:   "400",
			Type:      "Global",
			UpdatedAt: ptypes.TimestampNow(),
		},
	}

	// Test Add
	for _, d := range deployments {
		suite.NoError(suite.AddDeployment(d))
	}

	for _, d := range deployments {
		got, exists, err := suite.GetDeployment(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	// Test Update
	for _, d := range deployments {
		d.UpdatedAt = ptypes.TimestampNow()
		d.Version += "0"
	}

	for _, d := range deployments {
		suite.NoError(suite.UpdateDeployment(d))
	}

	for _, d := range deployments {
		got, exists, err := suite.GetDeployment(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	// Test Count
	count, err := suite.CountDeployments()
	suite.NoError(err)
	suite.Equal(len(deployments), count)

	// Test Remove
	for _, d := range deployments {
		suite.NoError(suite.RemoveDeployment(d.GetId()))
	}

	for _, d := range deployments {
		deployment, _, err := suite.GetDeployment(d.GetId())
		suite.NoError(err)
		suite.NotNil(deployment.GetTombstone())
	}
}
