package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	pollingInterval = 10 * time.Millisecond
	capTime         = 100 * time.Millisecond
	longTime        = 2 * time.Second
	backoff         = wait.Backoff{
		Duration: capTime,
		Factor:   1,
		Jitter:   0,
		Steps:    1,
		Cap:      capTime,
	}
)

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

func TestRetryTicker(t *testing.T) {
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
		t.Run(tcName, func(t *testing.T) {
			var done1, done2 Flag
			wait1 := 2 * capTime
			forcedErr := errors.New("forced")

			m := &testTickFun{}
			var ticker RetryTicker

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
				tickerBuilder := NewRetryTickerBuilder(m.f, longTime, backoff)
				ticker = tickerBuilder.OnTickSuccess(m.OnTickSuccess).OnTickError(m.OnTickError).Build()
				if !tc.forceError {
					m.On("OnTickSuccess", wait1).Once()
				} else {
					m.On("OnTickError", forcedErr).Once()
				}
				m.On("OnTickSuccess", longTime).Once()
			} else {
				ticker = NewRetryTicker(m.f, longTime, backoff)
			}

			ticker.Start()
			defer ticker.Stop()

			assert.True(t, PollWithTimeout(done1.Get, pollingInterval, capTime))
			if !tc.forceError {
				assert.True(t, PollWithTimeout(done2.Get, pollingInterval, wait1+capTime))
			} else {
				assert.True(t, PollWithTimeout(done2.Get, pollingInterval, backoff.Cap+capTime))
			}

			m.AssertExpectations(t)
		})
	}
}
