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
			handler = NewHandler()
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
			handler = NewHandler()
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

func (s *clusterIDSuite) Test_InitCertUpgrade() {
	initClusterID := "00000000-0000-0000-0000-000000000000"
	realClusterID := "real-cluster-id"

	s.Run("callback triggered when transitioning from init cert to real cluster ID", func() {
		handler := NewHandler()
		handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
			return initClusterID, nil
		}
		handler.isInitCertClusterID = func(id string) bool {
			return id == initClusterID
		}
		handler.getClusterID = func(value, _ string) (string, error) {
			return value, nil
		}

		// Trigger Get() to initialize isInitCertificate flag.
		// Get() will block waiting for Set() when using init cert, so don't wait for it.
		go func() {
			_ = handler.Get()
		}()
		// Brief sleep to let Get() reach the wait point and set isInitCertificate.
		time.Sleep(10 * time.Millisecond)

		callbackTriggered := false
		callbackDone := make(chan struct{})
		handler.RegisterInitCertUpgradeCallback(func() {
			callbackTriggered = true
			close(callbackDone)
		})

		handler.Set(realClusterID)

		select {
		case <-callbackDone:
			s.Assert().True(callbackTriggered, "callback should have been triggered")
		case <-time.After(2 * time.Second):
			s.Fail("timed out waiting for init cert upgrade callback")
		}

		s.Assert().False(handler.isInitCertificate, "isInitCertificate should be false after upgrade")
	})

	s.Run("callback NOT triggered when already using real certificate", func() {
		handler := NewHandler()
		handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
			return realClusterID, nil
		}
		handler.isInitCertClusterID = func(id string) bool {
			return id == initClusterID
		}
		handler.getClusterID = func(value, _ string) (string, error) {
			return value, nil
		}

		_ = handler.Get() // Non-blocking since not init cert.

		callbackTriggered := false
		callbackDone := make(chan struct{})
		handler.RegisterInitCertUpgradeCallback(func() {
			callbackTriggered = true
			close(callbackDone)
		})

		handler.Set(realClusterID)

		// Wait briefly to ensure callback does not fire.
		select {
		case <-callbackDone:
			s.Fail("callback should NOT have been triggered")
		case <-time.After(100 * time.Millisecond):
			// Expected - callback should not fire.
		}

		s.Assert().False(callbackTriggered, "callback should NOT have been triggered")
	})

	s.Run("callback NOT triggered on subsequent Set calls", func() {
		handler := NewHandler()
		handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
			return initClusterID, nil
		}
		handler.isInitCertClusterID = func(id string) bool {
			return id == initClusterID
		}
		handler.getClusterID = func(value, _ string) (string, error) {
			return value, nil
		}

		go func() {
			_ = handler.Get()
		}()
		time.Sleep(10 * time.Millisecond)

		callbackCount := 0
		callbackDone := make(chan struct{})
		handler.RegisterInitCertUpgradeCallback(func() {
			callbackCount++
			// Only close channel on first invocation.
			select {
			case <-callbackDone:
			default:
				close(callbackDone)
			}
		})

		// First Set should trigger callback.
		handler.Set(realClusterID)
		select {
		case <-callbackDone:
			s.Assert().Equal(1, callbackCount, "callback should be triggered once on first Set")
		case <-time.After(2 * time.Second):
			s.Fail("timed out waiting for init cert upgrade callback")
		}

		// Second Set should NOT trigger callback.
		handler.Set(realClusterID)
		// Wait briefly to ensure callback does not fire again.
		time.Sleep(100 * time.Millisecond)
		s.Assert().Equal(1, callbackCount, "callback should NOT be triggered on second Set")
	})

	s.Run("callback invoked immediately if registered after transition", func() {
		handler := NewHandler()
		handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
			return initClusterID, nil
		}
		handler.isInitCertClusterID = func(id string) bool {
			return id == initClusterID
		}
		handler.getClusterID = func(value, _ string) (string, error) {
			return value, nil
		}

		go func() {
			_ = handler.Get()
		}()
		time.Sleep(10 * time.Millisecond)

		// Set the real cluster ID to complete the transition.
		handler.Set(realClusterID)

		// Wait briefly to ensure transition is complete.
		time.Sleep(50 * time.Millisecond)

		// Register callback AFTER the transition has occurred.
		callbackTriggered := false
		callbackDone := make(chan struct{})
		handler.RegisterInitCertUpgradeCallback(func() {
			callbackTriggered = true
			close(callbackDone)
		})

		// Callback should be invoked immediately.
		select {
		case <-callbackDone:
			s.Assert().True(callbackTriggered, "callback should have been triggered immediately")
		case <-time.After(2 * time.Second):
			s.Fail("timed out waiting for immediate callback invocation")
		}
	})

	s.Run("callback handles nil gracefully", func() {
		handler := NewHandler()
		handler.parseClusterIDFromServiceCert = func(_ storage.ServiceType) (string, error) {
			return initClusterID, nil
		}
		handler.isInitCertClusterID = func(id string) bool {
			return id == initClusterID
		}
		handler.getClusterID = func(value, _ string) (string, error) {
			return value, nil
		}

		go func() {
			_ = handler.Get()
		}()
		time.Sleep(10 * time.Millisecond)

		// Don't register callback - test that nil callback doesn't panic.
		s.Assert().NotPanics(func() {
			handler.Set(realClusterID)
		})
	})
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
