package tests

import (
	"context"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type orScopeCheckerTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx context.Context
}

func (suite *orScopeCheckerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
}

func (suite *orScopeCheckerTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

/*
What do we want to test?

*/
