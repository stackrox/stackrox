package service

import (
	"context"
	"os"
	"testing"

	"github.com/boltdb/bolt"
	dDataStoreMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	networkFlowStore "github.com/stackrox/rox/central/networkflow/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

type NetworkGraphServiceTestSuite struct {
	suite.Suite
	db          *bolt.DB
	deployments *dDataStoreMocks.DataStore
	tested      Service
}

func (suite *NetworkGraphServiceTestSuite) SetupTest() {
	suite.deployments = &dDataStoreMocks.DataStore{}

	db, err := bolthelper.NewTemp("fun.db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db

	clusterStore := networkFlowStore.NewClusterStore(db)
	suite.tested = New(clusterStore, suite.deployments)
}

func (suite *NetworkGraphServiceTestSuite) TeardownSuite() {
	suite.db.Close()
	os.Remove(suite.db.Path())
}

func (suite *NetworkGraphServiceTestSuite) TestFailsIfClusterIsNotSet() {
	request := &v1.NetworkGraphRequest{}
	_, err := suite.tested.GetNetworkGraph((context.Context)(nil), request)
	suite.Error(err, "expected graph generation to fail since no cluster is specified")
}

func (suite *NetworkGraphServiceTestSuite) TestFailsIfSinceIsNotSet() {
	request := &v1.NetworkGraphRequest{ClusterId: "fake one"}
	_, err := suite.tested.GetNetworkGraph((context.Context)(nil), request)
	suite.Error(err, "expected graph generation to fail since no cluster is specified")
}

func TestNetworkGraph(t *testing.T) {
	suite.Run(t, new(NetworkGraphServiceTestSuite))
}
