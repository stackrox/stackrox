package clusterid

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/suite"
)

func TestClusterIDHandler(t *testing.T) {
	suite.Run(t, new(clusterIDSuite))
}

type clusterIDSuite struct {
	suite.Suite
}

func (s *clusterIDSuite) Test_SubsequentCallsToNewReturnNil() {
	s.Assert().NotNil(NewHandler())
	s.Assert().Panics(func() {
		_ = NewHandler()
	})
}

func (s *clusterIDSuite) Test_Get() {
	var handler *handlerImpl
	clusterID := "id"
	cases := map[string]struct {
		injectFakeCalls *funcWrapper
		shouldPanic     bool
		shouldWait      bool
		expectedID      string
	}{
		"error on parsing the id from the service cert should panic": {
			injectFakeCalls: newFuncWrapper(func() {
				handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
					return "", errors.New("some error")
				}
			}),
			shouldPanic: true,
		},
		"wait if is init ID": {
			injectFakeCalls: newFuncWrapper(func() {
				handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
					return clusterID, nil
				}
				handler.isInitCertClusterID = func(_ string) bool {
					return true
				}
			}),
			shouldWait: true,
		},
		"no wait if is not init ID": {
			injectFakeCalls: newFuncWrapper(func() {
				handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
					return clusterID, nil
				}
				handler.isInitCertClusterID = func(_ string) bool {
					return false
				}
			}),
			expectedID: clusterID,
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			handler = NewHandlerForTesting(s.T())
			tCase.injectFakeCalls.Run()
			if tCase.shouldPanic {
				s.Assert().Panics(func() {
					handler.Get()
				})
				return
			}
			if tCase.shouldWait {
				wg := sync.WaitGroup{}
				wg.Add(1)
				defer func() {
					// Call this to not leak the blocking Get call
					handler.clusterIDAvailable.Signal()
					wg.Wait()
				}()
				go func() {
					_ = handler.Get()
					wg.Done()
				}()
				select {
				case <-time.After(100 * time.Millisecond):
					return
				case <-handler.clusterIDAvailable.Done():
					s.FailNow("the call to Get should block")
				}
			}
			var actualID string
			defer func() {
				// Call this to not leak the blocking Get call in case of error
				handler.clusterIDAvailable.Signal()
			}()
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				actualID = handler.Get()
				wg.Done()
			}()
			select {
			case <-time.After(100 * time.Millisecond):
				s.FailNow("the call to Get should not block")
			case <-handler.clusterIDAvailable.Done():
			}
			wg.Wait()
			s.Assert().Equal(tCase.expectedID, actualID)
		})
	}
}

func (s *clusterIDSuite) Test_Set() {
	var handler *handlerImpl
	clusterID := "id"
	cases := map[string]struct {
		injectFakeCalls *funcWrapper
		clusterID       string
		shouldPanic     bool
	}{
		"error parsing the cluster ID should panic": {
			injectFakeCalls: newFuncWrapper(func() {
				handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
					return "", errors.New("some error")
				}
			}),
			shouldPanic: true,
		},
		"error getting the cluster ID should panic": {
			injectFakeCalls: newFuncWrapper(func() {
				handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
					return "", nil
				}
				handler.getClusterID = func(_, _ string) (string, error) {
					return "", errors.New("some error")
				}
			}),
			shouldPanic: true,
		},
		"a different ID should panic": {
			injectFakeCalls: newFuncWrapper(func() {
				handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
					return "", nil
				}
				handler.getClusterID = func(_, _ string) (string, error) {
					return "new-id", nil
				}
				handler.clusterID = "old-id"
				handler.clusterIDAvailable.Signal()
			}),
			shouldPanic: true,
		},
		"first ID should be set correctly": {
			injectFakeCalls: newFuncWrapper(func() {
				handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
					return "", nil
				}
				handler.getClusterID = func(_, _ string) (string, error) {
					return clusterID, nil
				}
			}),
		},
		"same ID should not panic": {
			injectFakeCalls: newFuncWrapper(func() {
				handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
					return "", nil
				}
				handler.getClusterID = func(_, _ string) (string, error) {
					return clusterID, nil
				}
				handler.clusterID = clusterID
				handler.clusterIDAvailable.Signal()
			}),
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			handler = NewHandlerForTesting(s.T())
			tCase.injectFakeCalls.Run()
			if tCase.shouldPanic {
				s.Assert().Panics(func() {
					handler.Set(tCase.clusterID)
				})
				return
			}
			handler.Set(clusterID)
			select {
			case <-time.After(100 * time.Millisecond):
				s.FailNow("cluster id available signal should be triggered")
			case <-handler.clusterIDAvailable.Done():
			}
			s.Assert().Equal(clusterID, handler.clusterID)
		})
	}
}

func newFuncWrapper(fn func()) *funcWrapper {
	return &funcWrapper{
		fn: fn,
	}
}

type funcWrapper struct {
	fn func()
}

func (w *funcWrapper) Run() {
	if w == nil || w.fn == nil {
		return
	}
	w.fn()
}
