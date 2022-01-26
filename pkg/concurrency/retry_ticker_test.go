package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	pollingInterval = 10 * time.Millisecond
	epsilonTime     = 100 * time.Millisecond
	longTime        = 2 * time.Second
	backoff         = wait.Backoff{
		Duration: epsilonTime,
		Factor:   1,
		Jitter:   0,
		Steps:    1,
		Cap:      epsilonTime,
	}
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(retryTickerSuite))
}

type retryTickerSuite struct {
	suite.Suite
}

type testTickFun struct {
	mock.Mock
}

func (f *testTickFun) f(ctx context.Context) (nextTimeToTick time.Duration, err error) {
	args := f.Called(ctx)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (f *testTickFun) OnTickSuccess(nextTimeToTick time.Duration) {
	f.Called(nextTimeToTick)
}

func (f *testTickFun) OnTickError(err error) {
	f.Called(err)
}

func (s *retryTickerSuite) TestRetryTicker() {
	testCases := map[string]struct {
		forceError       bool
		addEventHandlers bool
	}{
		"successWithEventHandlers":     {forceError: false, addEventHandlers: true},
		"successWithoutEventHandlers":  {forceError: false, addEventHandlers: false},
		"oneErrorWithEventHandlers":    {forceError: true, addEventHandlers: true},
		"oneErrorWithoutEventHandlers": {forceError: true, addEventHandlers: false},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			var done1, done2 Flag
			wait1 := 2 * epsilonTime
			forcedErr := errors.New("forced")

			m := &testTickFun{}
			ticker := NewRetryTicker(m.f, longTime, backoff)

			if !tc.forceError {
				m.On("f", mock.Anything).Return(wait1, nil).Run(func(args mock.Arguments) {
					done1.Set(true)
				}).Once()
			} else {
				m.On("f", mock.Anything).Return(time.Duration(0), forcedErr).Run(func(args mock.Arguments) {
					done1.Set(true)
				}).Once()
			}
			m.On("f", mock.Anything).Return(longTime, nil).Run(func(args mock.Arguments) {
				done2.Set(true)
			}).Once()
			if tc.addEventHandlers {
				ticker.OnTickSuccess = m.OnTickSuccess
				ticker.OnTickError = m.OnTickError
				if !tc.forceError {
					m.On("OnTickSuccess", wait1).Once()
				} else {
					m.On("OnTickError", forcedErr).Once()
				}
				m.On("OnTickSuccess", longTime).Once()
			}

			ticker.Start()
			defer ticker.Stop()

			s.True(PollWithTimeout(done1.Get, pollingInterval, epsilonTime))
			if !tc.forceError {
				s.True(PollWithTimeout(done2.Get, pollingInterval, wait1+epsilonTime))
			} else {
				s.True(PollWithTimeout(done2.Get, pollingInterval, backoff.Cap+epsilonTime))
			}

			m.AssertExpectations(s.T())
		})
	}
}
