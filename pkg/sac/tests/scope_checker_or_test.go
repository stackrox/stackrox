package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/mocks"
	"github.com/stretchr/testify/suite"
)

func TestOrScopeChecker(t *testing.T) {
	suite.Run(t, new(orScopeCheckerTestSuite))
}

type orScopeCheckerTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	scopeChecker1 *mocks.MockScopeChecker
	scopeChecker2 *mocks.MockScopeChecker

	orScopeChecker sac.ScopeChecker
}

func (suite *orScopeCheckerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.scopeChecker1 = mocks.NewMockScopeChecker(gomock.NewController(suite.T()))
	suite.scopeChecker2 = mocks.NewMockScopeChecker(gomock.NewController(suite.T()))

	suite.orScopeChecker = sac.NewOrScopeChecker(suite.scopeChecker1, suite.scopeChecker2)
}

func (suite *orScopeCheckerTestSuite) TestTryAllowed() {
	// 1. Expect true when at least 1 ScopeChecker returns true.
	suite.scopeChecker1.EXPECT().TryAllowed().Return(true)

	res := suite.orScopeChecker.TryAllowed()
	suite.Equal(true, res)

	// 2. Expect Deny when all ScopeChecker return false.
	suite.scopeChecker1.EXPECT().TryAllowed().Return(false)
	suite.scopeChecker2.EXPECT().TryAllowed().Return(false)

	res = suite.orScopeChecker.TryAllowed()
	suite.Equal(false, res)
}

func (suite *orScopeCheckerTestSuite) TestAllowed() {
	// 1. Expect True when at least 1 ScopeChecker returns true.
	suite.scopeChecker1.EXPECT().Allowed(gomock.Any()).Return(true, nil)

	allowed, err := suite.orScopeChecker.Allowed(context.Background())
	suite.True(allowed)
	suite.NoError(err)
	suite.Nil(err)

	// 2. Expect False when all ScopeCheckers return false.
	suite.scopeChecker1.EXPECT().Allowed(gomock.Any()).Return(false, nil)
	suite.scopeChecker2.EXPECT().Allowed(gomock.Any()).Return(false, nil)

	allowed, err = suite.orScopeChecker.Allowed(context.Background())
	suite.False(allowed)
	suite.NoError(err)
	suite.Nil(err)

	// 3. Expect an error and False when all ScopeCheckers return Deny and at least 1 returned an error.
	suite.scopeChecker1.EXPECT().Allowed(gomock.Any()).Return(false, nil)
	suite.scopeChecker2.EXPECT().Allowed(gomock.Any()).Return(false, errors.New("something happened"))

	allowed, err = suite.orScopeChecker.Allowed(context.Background())
	suite.False(allowed)
	suite.Error(err)
}

func (suite *orScopeCheckerTestSuite) TestAnyAllowed() {
	// 1. Expect True when at least 1 ScopeChecker returns true.
	suite.scopeChecker1.EXPECT().AnyAllowed(gomock.Any(), gomock.Any()).Return(true, nil)

	allowed, err := suite.orScopeChecker.AnyAllowed(context.Background(), nil)
	suite.NoError(err)
	suite.Nil(err)
	suite.True(allowed)

	// 2. Expect False when all ScopeCheckers return false.
	suite.scopeChecker1.EXPECT().AnyAllowed(gomock.Any(), gomock.Any()).Return(false, nil)
	suite.scopeChecker2.EXPECT().AnyAllowed(gomock.Any(), gomock.Any()).Return(false, nil)

	allowed, err = suite.orScopeChecker.AnyAllowed(context.Background(), nil)
	suite.NoError(err)
	suite.Nil(err)
	suite.False(allowed)

	// 3. Expect an error and False when all ScopeCheckers return Deny and at least 1 returned an error.
	suite.scopeChecker1.EXPECT().AnyAllowed(gomock.Any(), gomock.Any()).Return(false, nil)
	suite.scopeChecker2.EXPECT().AnyAllowed(gomock.Any(), gomock.Any()).Return(false,
		errors.New("something happened"))

	allowed, err = suite.orScopeChecker.AnyAllowed(context.Background(), nil)
	suite.Error(err)
	suite.False(allowed)
}

func (suite *orScopeCheckerTestSuite) TestAllAllowed() {
	// 1. Expect True when at least 1 ScopeChecker returns true.
	suite.scopeChecker1.EXPECT().AllAllowed(gomock.Any(), gomock.Any()).Return(true, nil)

	allowed, err := suite.orScopeChecker.AllAllowed(context.Background(), nil)
	suite.NoError(err)
	suite.Nil(err)
	suite.True(allowed)

	// 2. Expect False when all ScopeCheckers return false.
	suite.scopeChecker1.EXPECT().AllAllowed(gomock.Any(), gomock.Any()).Return(false, nil)
	suite.scopeChecker2.EXPECT().AllAllowed(gomock.Any(), gomock.Any()).Return(false, nil)

	allowed, err = suite.orScopeChecker.AllAllowed(context.Background(), nil)
	suite.NoError(err)
	suite.Nil(err)
	suite.False(allowed)

	// 3. Expect an error and False when all ScopeCheckers return Deny and at least 1 returned an error.
	suite.scopeChecker1.EXPECT().AllAllowed(gomock.Any(), gomock.Any()).Return(false, nil)
	suite.scopeChecker2.EXPECT().AllAllowed(gomock.Any(), gomock.Any()).Return(false, errors.New("something happened"))

	allowed, err = suite.orScopeChecker.AllAllowed(context.Background(), nil)
	suite.Error(err)
	suite.False(allowed)
}
