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
	capTime         = 500 * time.Millisecond
	longTime        = 5 * time.Second
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

func (f *testTickFun) doTick(ctx context.Context) (timeToNextTick time.Duration, err error) {
	args := f.Called(ctx)
	return args.Get(0).(time.Duration), args.Error(1)
}

func TestRetryTicker(t *testing.T) {
	testCases := map[string]struct {
		timeToSecondTick time.Duration
		firstErr         error
	}{
		"success":                 {timeToSecondTick: 2 * capTime, firstErr: nil},
		"with error should retry": {timeToSecondTick: 0, firstErr: errors.New("forced")},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			var done1, done2 Flag
			m := &testTickFun{}
			var ticker RetryTicker

			m.On("doTick", mock.Anything).Return(tc.timeToSecondTick, tc.firstErr).Run(func(args mock.Arguments) {
				done1.Set(true)
			}).Once()
			m.On("doTick", mock.Anything).Return(longTime, nil).Run(func(args mock.Arguments) {
				done2.Set(true)
			}).Once()
			ticker = NewRetryTicker(m.doTick, longTime, backoff)

			ticker.Start()
			defer ticker.Stop()

			// this should happen immediately, we add capTime to give some margin to make test more stable.
			assert.True(t, PollWithTimeout(done1.Get, pollingInterval, capTime))

			var expectedTimeToSecondAttempt time.Duration
			if tc.firstErr == nil {
				expectedTimeToSecondAttempt = tc.timeToSecondTick
			} else {
				expectedTimeToSecondAttempt = backoff.Cap
			}
			// we add capTime to give some margin to make test more stable.
			assert.True(t, PollWithTimeout(done2.Get, pollingInterval, expectedTimeToSecondAttempt+capTime))
		})
	}
}
