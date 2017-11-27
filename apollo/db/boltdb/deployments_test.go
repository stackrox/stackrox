package boltdb

import (
	"io/ioutil"
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
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
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		suite.FailNow("Failed to get temporary directory", err.Error())
	}
	db, err := MakeBoltDB(tmpDir)
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.BoltDB = db.(*BoltDB)
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

	// Test Remove
	for _, d := range deployments {
		suite.NoError(suite.RemoveDeployment(d.GetId()))
	}

	for _, d := range deployments {
		_, exists, err := suite.GetDeployment(d.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
