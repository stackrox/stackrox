package handler

import (
	"context"
	"testing"
	"time"

	dsMocks "github.com/stackrox/rox/central/notifications/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/notifications"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestNotificationsHandler(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(handlerTestSuite))
}

type handlerTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller
	mutex    sync.RWMutex

	datastore          *dsMocks.MockDataStore
	notificationStream notifications.Stream
	handler            *handlerImpl
}

func (s *handlerTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.datastore = dsMocks.NewMockDataStore(s.mockCtrl)
	s.notificationStream = notifications.NewStream()
	s.handler = newHandler(s.datastore, s.notificationStream).(*handlerImpl)
	flushInterval = 10 * time.Millisecond
	s.handler.Start()
}

func (s *handlerTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
	s.handler.Stop()
}

func (s *handlerTestSuite) TestConsumeNotifications() {
	notification := &storage.Notification{
		Level:   storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
		Message: "message",
		Type:    storage.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Hint:    "hint",
		Domain:  "domain",
	}

	addCalled := false
	addSetCalledFn := func(ctx context.Context, notification *storage.Notification) {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		addCalled = true
	}
	addCalledFn := func() bool {
		s.mutex.RLock()
		defer s.mutex.RUnlock()
		return addCalled
	}
	s.datastore.EXPECT().AddNotification(notificationWriteCtx, notification).Do(addSetCalledFn)

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
	s.datastore.EXPECT().Flush(notificationWriteCtx).MinTimes(1).Do(flushSetCalledFn)

	err := s.notificationStream.Produce(notification)
	s.Require().NoError(err)

	s.Eventually(addCalledFn, 100*time.Millisecond, 10*time.Millisecond)
	s.Eventually(flushCalledFn, 100*time.Millisecond, 10*time.Millisecond)
}
