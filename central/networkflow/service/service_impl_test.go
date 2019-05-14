package service

import (
	"context"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/golang/mock/gomock"
	dDSMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	nfDSMocks "github.com/stackrox/rox/central/networkflow/datastore/mocks"
	npDSMocks "github.com/stackrox/rox/central/networkpolicies/graph/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestNetworkGraph(t *testing.T) {
	suite.Run(t, new(NetworkGraphServiceTestSuite))
}

type NetworkGraphServiceTestSuite struct {
	suite.Suite
	db          *bolt.DB
	deployments *dDSMocks.MockDataStore
	evaluator   *npDSMocks.MockEvaluator
	tested      Service

	mockCtrl *gomock.Controller
}

func (suite *NetworkGraphServiceTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.deployments = dDSMocks.NewMockDataStore(suite.mockCtrl)

	db, err := bolthelper.NewTemp("fun.db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db

	clusterStore := nfDSMocks.NewMockClusterDataStore(suite.mockCtrl)
	suite.evaluator = npDSMocks.NewMockEvaluator(suite.mockCtrl)

	suite.tested = New(clusterStore, suite.deployments, suite.evaluator)
}

func (suite *NetworkGraphServiceTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)

	suite.mockCtrl.Finish()
}

func (suite *NetworkGraphServiceTestSuite) TestFailsIfClusterIsNotSet() {
	request := &v1.NetworkGraphRequest{}
	_, err := suite.tested.GetNetworkGraph(context.TODO(), request)
	suite.Error(err, "expected graph generation to fail since no cluster is specified")
}
