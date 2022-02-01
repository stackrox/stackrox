package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	testTimeout = 1 * time.Second
	longTime    = 5 * time.Second
	capTime     = 100 * time.Millisecond
	backoff     = wait.Backoff{
		Duration: capTime,
		Factor:   1,
		Jitter:   0,
		Steps:    2,
		Cap:      capTime,
	}
)

type testTickFunc struct {
	mock.Mock
}

func (f *testTickFunc) doTick(ctx context.Context) (timeToNextTick time.Duration, err error) {
	args := f.Called(ctx)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (f *testTickFunc) Step() time.Duration {
	f.Called()
	return 0
}

type afterFuncSpy struct {
	mock.Mock
}

func (f *afterFuncSpy) afterFunc(d time.Duration, fn func()) *time.Timer {
	f.Called(d)
	return time.AfterFunc(d, fn)
}

func TestRetryTicker(t *testing.T) {
	testCases := map[string]struct {
		timeToSecondTick time.Duration
		firstErr         error
	}{
		"success":                 {timeToSecondTick: capTime, firstErr: nil},
		"with error should retry": {timeToSecondTick: 0, firstErr: errors.New("forced")},
	}
	for tcName, tc := range testCases {
		t.Run(tcName, func(t *testing.T) {
			doneErrSig := NewErrorSignal()
			mockFunc := &testTickFunc{}
			schedulerSpy := &afterFuncSpy{}

			mockFunc.On("doTick", mock.Anything).Return(tc.timeToSecondTick, tc.firstErr).Once()
			mockFunc.On("doTick", mock.Anything).Return(longTime, nil).Run(func(args mock.Arguments) {
				doneErrSig.Signal()
			}).Once()
			mockFunc.On("doTick", mock.Anything).Return(longTime, nil).Maybe()

			schedulerSpy.On("afterFunc", time.Duration(0), mock.Anything).Return(nil).Once()
			if tc.firstErr == nil {
				schedulerSpy.On("afterFunc", tc.timeToSecondTick, mock.Anything).Return(nil).Once()
			} else {
				schedulerSpy.On("afterFunc", backoff.Duration, mock.Anything).Return(nil).Once()
			}
			schedulerSpy.On("afterFunc", longTime, mock.Anything).Return(nil).Maybe()

			newTicker := NewRetryTicker(mockFunc.doTick, longTime, backoff)
			require.IsType(t, &retryTickerImpl{}, newTicker)
			ticker := newTicker.(*retryTickerImpl)
			ticker.scheduler = schedulerSpy.afterFunc

			ticker.Start()
			defer ticker.Stop()

			_, ok := doneErrSig.WaitWithTimeout(testTimeout)
			assert.True(t, ok)
			mockFunc.AssertExpectations(t)
			schedulerSpy.AssertExpectations(t)
		})
	}
}
