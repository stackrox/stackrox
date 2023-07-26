package contextprovider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	contextKey         = "context-key"
	contextValuePrefix = "context-value"
)

type contextProviderSuite struct {
	suite.Suite
}

func TestContextProvider(t *testing.T) {
	suite.Run(t, new(contextProviderSuite))
}

func (s *contextProviderSuite) TestGetContextBeforeCentralReachable() {
	provider := s.createContextProvider(newTestContextFnHelper(contextValuePrefix).contextWithValue())

	ctx := provider.GetContext()
	s.Assert().Equal(fmt.Sprintf("%s-%d", contextValuePrefix, 1), ctx.Value(contextKey))
}

func (s *contextProviderSuite) TestGetContextAfterCentralReachable() {
	provider := s.createContextProvider(newTestContextFnHelper(contextValuePrefix).contextWithValue())

	s.Assert().Equal(fmt.Sprintf("%s-%d", contextValuePrefix, 1), provider.GetContext().Value(contextKey))
	provider.Notify(common.SensorComponentEventCentralReachable)
	// After central is reachable the context should be the same
	s.Assert().Equal(fmt.Sprintf("%s-%d", contextValuePrefix, 1), provider.GetContext().Value(contextKey))
}

func (s *contextProviderSuite) TestGetContextDifferentContextAfterReconnect() {
	provider := s.createContextProvider(newTestContextFnHelper(contextValuePrefix).contextWithValue())

	provider.Notify(common.SensorComponentEventCentralReachable)
	s.Assert().Equal(fmt.Sprintf("%s-%d", contextValuePrefix, 1), provider.GetContext().Value(contextKey))
	provider.Notify(common.SensorComponentEventOfflineMode)
	provider.Notify(common.SensorComponentEventCentralReachable)
	// After central is reachable again the context should be the different
	s.Assert().Equal(fmt.Sprintf("%s-%d", contextValuePrefix, 2), provider.GetContext().Value(contextKey))
}

func (s *contextProviderSuite) TestGetContextCancelInOfflineMode() {
	provider := s.createContextProvider(newTestContextFnHelper(contextValuePrefix).contextWithValue())

	ctx := provider.GetContext()
	provider.Notify(common.SensorComponentEventCentralReachable)
	s.Assert().Equal(fmt.Sprintf("%s-%d", contextValuePrefix, 1), ctx.Value(contextKey))
	select {
	case <-ctx.Done():
		s.Fail("the context should not be cancelled")
	case <-time.After(10 * time.Millisecond):
		break
	}
	provider.Notify(common.SensorComponentEventOfflineMode)
	select {
	case <-ctx.Done():
		break
	case <-time.After(5 * time.Second):
		s.Fail("the context should be cancelled")
	}
}

func (s *contextProviderSuite) TestGetContextCallFromMultipleGoroutines() {
	provider := s.createContextProvider(newTestContextFnHelper(contextValuePrefix).contextWithValue())
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.Assert().Equal(fmt.Sprintf("%s-%d", contextValuePrefix, 1), provider.GetContext().Value(contextKey))
	}()
	go func() {
		defer wg.Done()
		s.Assert().Equal(fmt.Sprintf("%s-%d", contextValuePrefix, 1), provider.GetContext().Value(contextKey))
	}()
	wg.Wait()
}

func (s *contextProviderSuite) TestOrderOfComponentsMatter() {
	provider := s.createContextProvider(newTestContextFnHelper(contextValuePrefix).contextWithValue())
	components := []notifiableComponent{
		// The provider needs to be the first in getting notify.
		// Otherwise, other components would get and old context on CentralReachable.
		provider,
		&fakeComponent{
			provider: provider,
		},
		&fakeComponent{
			provider: provider,
		},
	}
	states := []struct {
		event common.SensorComponentEvent
		value int
	}{
		{
			event: common.SensorComponentEventCentralReachable,
			value: 1,
		},
		{
			event: common.SensorComponentEventOfflineMode,
			value: 1,
		},
		{
			event: common.SensorComponentEventCentralReachable,
			value: 2,
		},
	}
	for _, state := range states {
		for _, component := range components {
			component.Notify(state.event)
		}
		for _, component := range components {
			if testable, ok := component.(*fakeComponent); ok {
				testable.assertContextValue(s.T(), contextKey, fmt.Sprintf("%s-%d", contextValuePrefix, state.value))
				testable.assertCancelContext(s.T(), state.event)
			}
		}
	}
}

type notifiableComponent interface {
	Notify(common.SensorComponentEvent)
}

type fakeComponent struct {
	provider ContextProvider
	ctx      context.Context
}

func (f *fakeComponent) Notify(event common.SensorComponentEvent) {
	switch event {
	case common.SensorComponentEventCentralReachable:
		f.ctx = f.provider.GetContext()
	case common.SensorComponentEventOfflineMode:
	}
}

func (f *fakeComponent) assertContextValue(t *testing.T, key, value string) {
	assert.Equal(t, value, f.ctx.Value(key))
}

func (f *fakeComponent) assertCancelContext(t *testing.T, event common.SensorComponentEvent) {
	switch event {
	case common.SensorComponentEventCentralReachable:
		select {
		case <-f.ctx.Done():
			t.Error("the context should not be cancelled")
			return
		case <-time.After(10 * time.Millisecond):
			return
		}
	case common.SensorComponentEventOfflineMode:
		select {
		case <-f.ctx.Done():
			return
		case <-time.After(5 * time.Second):
			t.Error("the context should be cancelled")
			return
		}
	}
	if event == common.SensorComponentEventOfflineMode {
	}
}

type testContextFnHelper struct {
	value         string
	contextNumber int
}

func newTestContextFnHelper(value string) *testContextFnHelper {
	return &testContextFnHelper{
		value:         value,
		contextNumber: 0,
	}
}

func (h *testContextFnHelper) contextWithValue() func() (context.Context, func()) {
	return func() (context.Context, func()) {
		h.contextNumber++
		return context.WithCancel(context.WithValue(context.Background(), contextKey, fmt.Sprintf("%s-%d", h.value, h.contextNumber)))
	}
}

func (s *contextProviderSuite) createContextProvider(contextFn func() (context.Context, func())) *contextProviderImpl {
	provider, ok := NewContextProvider().(*contextProviderImpl)
	s.Assert().True(ok)
	provider.init(contextFn)
	s.Assert().NoError(provider.Start())
	provider.newContextFn = contextFn
	return provider
}
