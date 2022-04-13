package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/sac/mocks"
	"github.com/stretchr/testify/suite"
)

type scopeCheckerTestSuite struct {
	suite.Suite

	ctx context.Context

	mockCtrl *gomock.Controller
	mockSCC  *mocks.MockScopeCheckerCore
}

func TestAllowed(t *testing.T) {
	suite.Run(t, new(scopeCheckerTestSuite))
}

func (s *scopeCheckerTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockSCC = mocks.NewMockScopeCheckerCore(s.mockCtrl)
	s.ctx = context.Background()
}

func (s *scopeCheckerTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *scopeCheckerTestSuite) TestAllowed_UnknownThenAllow() {
	result := Unknown
	s.mockSCC.EXPECT().TryAllowed().AnyTimes().DoAndReturn(func() TryAllowedResult {
		return result
	})
	s.mockSCC.EXPECT().PerformChecks(s.ctx).Times(1).DoAndReturn(func(context.Context) error {
		result = Allow
		return nil
	})

	ok, err := NewScopeChecker(s.mockSCC).Allowed(s.ctx)
	s.True(ok)
	s.NoError(err)
}

func (s *scopeCheckerTestSuite) TestAllowed_UnknownThenDeny() {
	result := Unknown
	s.mockSCC.EXPECT().TryAllowed().AnyTimes().DoAndReturn(func() TryAllowedResult {
		return result
	})
	s.mockSCC.EXPECT().PerformChecks(s.ctx).Times(1).DoAndReturn(func(context.Context) error {
		result = Deny
		return nil
	})

	ok, err := NewScopeChecker(s.mockSCC).Allowed(s.ctx)
	s.False(ok)
	s.NoError(err)
}

func (s *scopeCheckerTestSuite) UnknownThenError() {
	result := Unknown
	s.mockSCC.EXPECT().TryAllowed().AnyTimes().DoAndReturn(func() TryAllowedResult {
		return result
	})
	s.mockSCC.EXPECT().PerformChecks(s.ctx).Times(1).Return(errors.New("unknown error"))

	_, err := NewScopeChecker(s.mockSCC).Allowed(s.ctx)
	s.Error(err)
}
