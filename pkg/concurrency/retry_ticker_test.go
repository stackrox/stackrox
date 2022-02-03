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

func TestRetryTickerCallsTickFunction(t *testing.T) {
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
			ticker := newRetryTicker(t, mockFunc.doTick)
			ticker.scheduler = schedulerSpy.afterFunc

			mockFunc.On("doTick", mock.Anything).Return(tc.timeToSecondTick, tc.firstErr).Once()
			mockFunc.On("doTick", mock.Anything).Return(longTime, nil).Run(func(args mock.Arguments) {
				ticker.Stop()
				doneErrSig.Signal()
			}).Once()
			schedulerSpy.On("afterFunc", time.Duration(0), mock.Anything).Return(nil).Once()
			if tc.firstErr == nil {
				schedulerSpy.On("afterFunc", tc.timeToSecondTick, mock.Anything).Return(nil).Once()
			} else {
				schedulerSpy.On("afterFunc", backoff.Duration, mock.Anything).Return(nil).Once()
			}

			ticker.Start()
			defer ticker.Stop()

			_, ok := doneErrSig.WaitWithTimeout(testTimeout)
			assert.True(t, ok, "timeout exceeded")
			mockFunc.AssertExpectations(t)
			schedulerSpy.AssertExpectations(t)
		})
	}
}

func TestRetryTickerStop(t *testing.T) {
	firsTickErrSig := NewErrorSignal()
	stopErrSig := NewErrorSignal()
	ticker := newRetryTicker(t, func(ctx context.Context) (timeToNextTick time.Duration, err error) {
		firsTickErrSig.Signal()
		_, ok := stopErrSig.WaitWithTimeout(testTimeout)
		require.True(t, ok)
		return capTime, nil
	})

	ticker.Start()
	_, ok := firsTickErrSig.WaitWithTimeout(testTimeout)
	require.True(t, ok, "timeout exceeded")
	ticker.Stop()
	stopErrSig.Signal()

	// ensure `ticker.scheduleTick` does not schedule a new timer after stopping the ticker
	time.Sleep(capTime)
	assert.Nil(t, ticker.getTickTimer())
}

func newRetryTicker(t *testing.T, doFunc tickFunc) *retryTickerImpl {
	ticker := NewRetryTicker(doFunc, longTime, backoff)
	require.IsType(t, &retryTickerImpl{}, ticker)
	return ticker.(*retryTickerImpl)
}
