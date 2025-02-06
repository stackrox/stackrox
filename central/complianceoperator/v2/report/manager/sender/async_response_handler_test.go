package sender

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestAsyncResponseHandler(t *testing.T) {
	suite.Run(t, new(asyncResponseHandlerSuite))
}

type asyncResponseHandlerSuite struct {
	suite.Suite
}

func (s *asyncResponseHandlerSuite) Test_New() {
	noopOnSuccess := func(_ struct{}) error {
		return nil
	}
	noopOnError := func() {}
	resC := make(chan struct{})
	defer close(resC)
	s.Run("nil onSuccess callback", func() {
		h, err := NewAsyncResponseHandler[struct{}](nil, noopOnError, resC)
		assert.Error(s.T(), err)
		assert.True(s.T(), errors.Is(err, ErrInvalidInput))
		assert.Nil(s.T(), h)
	})
	s.Run("nil onError callback", func() {
		h, err := NewAsyncResponseHandler[struct{}](noopOnSuccess, nil, resC)
		assert.Error(s.T(), err)
		assert.True(s.T(), errors.Is(err, ErrInvalidInput))
		assert.Nil(s.T(), h)
	})
	s.Run("nil responseC", func() {
		h, err := NewAsyncResponseHandler[struct{}](noopOnSuccess, noopOnError, nil)
		assert.Error(s.T(), err)
		assert.True(s.T(), errors.Is(err, ErrInvalidInput))
		assert.Nil(s.T(), h)
	})
	s.Run("correct creation", func() {
		h, err := NewAsyncResponseHandler[struct{}](noopOnSuccess, noopOnError, resC)
		assert.NoError(s.T(), err)
		assert.NotNil(s.T(), h)
	})
}

func (s *asyncResponseHandlerSuite) Test_Run() {
	noopOnSuccess := func(_ *testResponse) error {
		return nil
	}
	noopOnError := func() {}
	cases := map[string]struct {
		numCallsToWait int
		response       *testResponse
		onSuccess      func(*concurrency.WaitGroup) func(*testResponse) error
		onError        func(*concurrency.WaitGroup) func()
		stopFn         func(AsyncResponseHandler[*testResponse])
		closeEarly     bool
	}{
		"on success": {
			numCallsToWait: 1,
			response:       &testResponse{},
			onSuccess:      testOnSuccess,
			onError: func(_ *concurrency.WaitGroup) func() {
				return noopOnError
			},
		},
		"on error": {
			numCallsToWait: 2,
			response: &testResponse{
				err: errors.New("error"),
			},
			onSuccess: testOnSuccess,
			onError:   testOnError,
		},
		"on stop": {
			numCallsToWait: 1,
			onSuccess: func(_ *concurrency.WaitGroup) func(*testResponse) error {
				return noopOnSuccess
			},
			onError: testOnError,
			stopFn: func(h AsyncResponseHandler[*testResponse]) {
				h.Stop()
			},
		},
		"on close chan": {
			numCallsToWait: 1,
			onSuccess: func(_ *concurrency.WaitGroup) func(*testResponse) error {
				return noopOnSuccess
			},
			onError:    testOnError,
			closeEarly: true,
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			resC := make(chan *testResponse)
			defer func() {
				if !tCase.closeEarly {
					close(resC)
				}
			}()
			wg := concurrency.NewWaitGroup(tCase.numCallsToWait)
			h, err := NewAsyncResponseHandler[*testResponse](tCase.onSuccess(&wg), tCase.onError(&wg), resC)
			require.NoError(s.T(), err)
			h.Start()

			go func() {
				if tCase.response != nil {
					resC <- tCase.response
				}
			}()
			if tCase.stopFn != nil {
				tCase.stopFn(h)
			}
			if tCase.closeEarly {
				close(resC)
			}

			handleWaitGroup(s.T(), &wg, 500*time.Millisecond, "the callback to be triggered")
			handlerImp, ok := h.(*asyncResponseHandlerImpl[*testResponse])
			require.True(s.T(), ok)
			select {
			case <-time.After(10 * time.Millisecond):
				s.T().Error("timeout waiting for the handler to stop")
				s.T().FailNow()
			case <-handlerImp.stopper.Client().Stopped().Done():
			}
		})
	}
}

type testResponse struct {
	err error
}

func testOnSuccess(wg *concurrency.WaitGroup) func(*testResponse) error {
	return func(res *testResponse) error {
		wg.Add(-1)
		return res.err
	}
}

func testOnError(wg *concurrency.WaitGroup) func() {
	return func() {
		wg.Add(-1)
	}
}

func handleWaitGroup(t *testing.T, wg *concurrency.WaitGroup, timeout time.Duration, msg string) {
	select {
	case <-time.After(timeout):
		t.Errorf("timeout waiting for %s", msg)
		t.Fail()
	case <-wg.Done():
	}
}
