package tests

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/mocks"
	"github.com/stretchr/testify/suite"
)

func TestScopeCheckerCore(t *testing.T) {
	suite.Run(t, new(scopeCheckerCoreTestSuite))
}

type scopeCheckerCoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	reqTracker *mocks.MockScopeRequestTracker
	ctx        context.Context
}

func (suite *scopeCheckerCoreTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.reqTracker = mocks.NewMockScopeRequestTracker(suite.mockCtrl)
}

func (suite *scopeCheckerCoreTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *scopeCheckerCoreTestSuite) getSCC() sac.ScopeCheckerCore {
	return sac.NewScopeCheckerCore(payload.AccessScope{}, suite.reqTracker)
}

func (suite *scopeCheckerCoreTestSuite) TestSubScopeChecker() {
	scc := suite.getSCC()

	scopeKeyOne := sac.AccessModeScopeKey(storage.Access_READ_ACCESS)
	subScope := scc.SubScopeChecker(scopeKeyOne)
	subScopeEqual := scc.SubScopeChecker(scopeKeyOne)
	suite.Equal(subScope, subScopeEqual)
	scopeKeyTwo := sac.AccessModeScopeKey(storage.Access_READ_WRITE_ACCESS)
	subScopeNotEqual := scc.SubScopeChecker(scopeKeyTwo)
	suite.NotEqual(subScope, subScopeNotEqual)
}

func (suite *scopeCheckerCoreTestSuite) TestTryAllowed() {
	scc := suite.getSCC()
	suite.reqTracker.EXPECT().AddRequested(scc).Return()
	result := scc.TryAllowed()
	suite.Equal(sac.Unknown, result)
	// Calling TryAllowed twice should only call reqTracker.AddRequested once
	scc.TryAllowed()
}

func (suite *scopeCheckerCoreTestSuite) TestPerformChecks() {
	scc := suite.getSCC()
	suite.reqTracker.EXPECT().PerformChecks(suite.ctx).Return(nil)
	err := scc.PerformChecks(suite.ctx)
	suite.NoError(err)
}
