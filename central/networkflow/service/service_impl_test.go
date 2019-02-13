package service

import (
	"context"
	"os"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/golang/mock/gomock"
	dDataStoreMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	networkFlowStore "github.com/stackrox/rox/central/networkflow/store/mocks"
	npGraphMocks "github.com/stackrox/rox/central/networkpolicies/graph/mocks"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

type NetworkGraphServiceTestSuite struct {
	suite.Suite
	db          *bolt.DB
	deployments *dDataStoreMocks.MockDataStore
	evaluator   *npGraphMocks.MockEvaluator
	tested      Service

	mockCtrl *gomock.Controller
}

func (suite *NetworkGraphServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.deployments = dDataStoreMocks.NewMockDataStore(suite.mockCtrl)

	db, err := bolthelper.NewTemp("fun.db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db

	clusterStore := networkFlowStore.NewMockClusterStore(suite.mockCtrl)
	suite.evaluator = npGraphMocks.NewMockEvaluator(suite.mockCtrl)

	suite.tested = New(clusterStore, suite.deployments, suite.evaluator)
}

func (suite *NetworkGraphServiceTestSuite) TearDownTest() {
	suite.db.Close()
	os.Remove(suite.db.Path())

	suite.mockCtrl.Finish()
}

func (suite *NetworkGraphServiceTestSuite) TestFailsIfClusterIsNotSet() {
	request := &v1.NetworkGraphRequest{}
	_, err := suite.tested.GetNetworkGraph((context.Context)(nil), request)
	suite.Error(err, "expected graph generation to fail since no cluster is specified")
}

func TestNetworkGraph(t *testing.T) {
	suite.Run(t, new(NetworkGraphServiceTestSuite))
}
