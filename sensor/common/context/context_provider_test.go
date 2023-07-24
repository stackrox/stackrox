package contextprovider

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/suite"
)

type contextProviderSuite struct {
	suite.Suite
}

func TestContextProvider(t *testing.T) {
	suite.Run(t, new(contextProviderSuite))
}

func (s *contextProviderSuite) TestGetContextBlockUntilCentralReachable() {
	provider := s.createContextProvider()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		s.Assert().NotNil(provider.GetContext())
		s.Assert().True(provider.centralReachable.IsDone(), "the context should not be returned until central is reachable")
		wg.Done()
	}()

	provider.Notify(common.SensorComponentEventCentralReachable)
	wg.Wait()
}

func (s *contextProviderSuite) TestGetContextBlocksUntilStopped() {
	provider := s.createContextProvider()

	provider.Notify(common.SensorComponentEventCentralReachable)
	provider.Notify(common.SensorComponentEventOfflineMode)
	ch := make(chan struct{})
	defer close(ch)
	go func() {
		s.Assert().Nil(provider.GetContext())
		s.Assert().False(provider.centralReachable.IsDone())
		ch <- struct{}{}
	}()
	select {
	case <-time.After(time.Millisecond):
		break
	case <-ch:
		s.Fail("context should not returned if central is not reachable")
		break
	}
	provider.Stop(nil)
	select {
	case <-time.After(5 * time.Second):
		s.Fail("timeout waiting for the component to be stopped")
	case <-ch:
		break
	}
}

func (s *contextProviderSuite) TestOfflineMode() {
	provider := s.createContextProvider()

	s.Assert().Nil(provider.sensorContext)
	s.Assert().Nil(provider.cancelContextFn)
	provider.Notify(common.SensorComponentEventCentralReachable)
	oldCtx := provider.sensorContext
	s.Assert().NotNil(oldCtx)
	s.Assert().NotNil(provider.cancelContextFn)
	provider.Notify(common.SensorComponentEventOfflineMode)
	select {
	case <-oldCtx.Done():
		break
	case <-time.After(5 * time.Second):
		s.Fail("timeout waiting for the context to be canceled")
	}
	provider.Notify(common.SensorComponentEventCentralReachable)
	s.Assert().NotEqual(oldCtx, provider.sensorContext)
	s.Assert().NotNil(provider.cancelContextFn)
	select {
	case <-provider.sensorContext.Done():
		s.Fail("the context should not be canceled when central is reachable")
	case <-time.After(time.Millisecond):
		break
	}
}

func (s *contextProviderSuite) createContextProvider() *contextProviderImpl {
	provider, ok := NewContextProvider().(*contextProviderImpl)
	s.Assert().True(ok)
	s.Assert().NoError(provider.Start())
	return provider
}
