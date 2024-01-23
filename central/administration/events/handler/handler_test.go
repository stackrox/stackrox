package handler

import (
	"context"
	"testing"
	"time"

	dsMocks "github.com/stackrox/rox/central/administration/events/datastore/mocks"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/administration/events/stream"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestEventsHandler(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(handlerTestSuite))
}

type handlerTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller
	mutex    sync.RWMutex

	datastore   *dsMocks.MockDataStore
	eventStream events.Stream
	handler     *handlerImpl
}

func (s *handlerTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.datastore = dsMocks.NewMockDataStore(s.mockCtrl)
	s.eventStream = stream.GetStreamForTesting(s.T())
	s.handler = newHandler(s.datastore, s.eventStream).(*handlerImpl)
	flushInterval = 10 * time.Millisecond
}

func (s *handlerTestSuite) TearDownTest() {
	s.handler.Stop()
}

func (s *handlerTestSuite) TestConsumeEvents() {
	event := fixtures.GetAdministrationEvent()

	addCalled := false
	addSetCalledFn := func(ctx context.Context, event *events.AdministrationEvent) {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		addCalled = true
	}
	addCalledFn := func() bool {
		s.mutex.RLock()
		defer s.mutex.RUnlock()
		return addCalled
	}
	s.datastore.EXPECT().AddEvent(s.handler.eventWriteCtx, event).Do(addSetCalledFn)

	flushCalled := false
	flushSetCalledFn := func(ctx context.Context) {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		flushCalled = true
	}
	flushCalledFn := func() bool {
		s.mutex.RLock()
		defer s.mutex.RUnlock()
		return flushCalled
	}
	s.datastore.EXPECT().Flush(s.handler.eventWriteCtx).MinTimes(1).Do(flushSetCalledFn)

	s.handler.Start()
	s.eventStream.Produce(event)

	s.Eventually(addCalledFn, 100*time.Millisecond, 10*time.Millisecond)
	s.Eventually(flushCalledFn, 100*time.Millisecond, 10*time.Millisecond)
}
