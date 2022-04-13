package sac

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/default-authz-plugin/pkg/payload"
	clusterMocks "github.com/stackrox/stackrox/central/cluster/datastore/mocks"
	"github.com/stackrox/stackrox/pkg/sac"
	clientMocks "github.com/stackrox/stackrox/pkg/sac/client/mocks"
	"github.com/stretchr/testify/suite"
)

func TestScopeRequestTracker(t *testing.T) {
	suite.Run(t, new(scopeRequestTrackerTestSuite))
}

type scopeRequestTrackerTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	client        *clientMocks.MockClient
	clusterGetter *clusterMocks.MockDataStore
	ctx           context.Context
}

func (suite *scopeRequestTrackerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.client = clientMocks.NewMockClient(suite.mockCtrl)
	suite.clusterGetter = clusterMocks.NewMockDataStore(suite.mockCtrl)

	suite.ctx = context.Background()
}

func (suite *scopeRequestTrackerTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *scopeRequestTrackerTestSuite) makeReqTracker() *ScopeRequestTrackerImpl {
	tracker := &ScopeRequestTrackerImpl{
		client:           suite.client,
		clusterDataStore: suite.clusterGetter,
		requestList:      []sac.ScopeRequest{},
		runnerChannel:    make(chan struct{}, 1),
		principal:        &payload.Principal{},
	}
	tracker.runnerChannel <- struct{}{}
	return tracker
}

func (suite *scopeRequestTrackerTestSuite) TestScopeRequestTracker() {
	scopeCheckers := []sac.ScopeRequest{&sac.ScopeCheckerCoreImpl{}, &sac.ScopeCheckerCoreImpl{}}
	tracker := suite.makeReqTracker()
	tracker.AddRequested(scopeCheckers...)
	gotRequests := tracker.getAndClearPendingRequests()
	suite.ElementsMatch(scopeCheckers, gotRequests)

	noRequests := tracker.getAndClearPendingRequests()
	suite.ElementsMatch([]sac.ScopeRequest{}, noRequests)
}

func (suite *scopeRequestTrackerTestSuite) TestPerformChecks() {
	tracker := suite.makeReqTracker()
	allowedScope := payload.AccessScope{Verb: "Allow"}
	deniedScope := payload.AccessScope{Verb: "Deny"}
	testAllowed := sac.NewScopeCheckerCore(allowedScope, tracker).(*sac.ScopeCheckerCoreImpl)
	testDenied := sac.NewScopeCheckerCore(deniedScope, tracker).(*sac.ScopeCheckerCoreImpl)
	suite.client.EXPECT().ForUser(gomock.Any(), gomock.Any(), gomock.Any()).Return([]payload.AccessScope{allowedScope}, []payload.AccessScope{deniedScope}, nil)

	tracker.AddRequested(testAllowed, testDenied)
	err := tracker.PerformChecks(suite.ctx)
	suite.NoError(err)
	suite.Equal(testAllowed.TryAllowed(), sac.Allow)
	suite.Equal(testDenied.TryAllowed(), sac.Deny)
}
