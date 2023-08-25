package writer

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	storeMocks "github.com/stackrox/rox/central/notifications/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var errFake = errors.New("fake error")

func TestNotificationsWriter(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(writerTestSuite))
}

type writerTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx    context.Context
	store  *storeMocks.MockStore
	writer Writer
}

func (s *writerTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.ctx = context.Background()
	s.store = storeMocks.NewMockStore(s.mockCtrl)
	s.writer = New(s.store)
}

func (s *writerTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *writerTestSuite) TestWriteNotification_Success() {
	notification := &storage.Notification{
		Level:        storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
		Message:      "message",
		Type:         storage.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Hint:         "hint",
		Domain:       "domain",
		CreatedAt:    protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurred: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}
	enrichedNotification := &storage.Notification{
		Id:           "0925514f-3a33-5931-b431-756406e1a008",
		Level:        storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
		Message:      "message",
		Type:         storage.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Hint:         "hint",
		Domain:       "domain",
		Occurrences:  1,
		CreatedAt:    protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurred: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}

	s.store.EXPECT().Get(s.ctx, enrichedNotification.GetId()).Return(nil, false, nil)
	err := s.writer.Upsert(s.ctx, notification)
	s.Require().NoError(err)

	s.store.EXPECT().UpsertMany(s.ctx, []*storage.Notification{enrichedNotification}).Return(nil)
	err = s.writer.Flush(s.ctx)
	s.Require().NoError(err)
}

func (s *writerTestSuite) TestWriteNotification_MergeWithBuffer() {
	id := "0925514f-3a33-5931-b431-756406e1a008"
	notificationBase := &storage.Notification{
		Id:           id,
		Level:        storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
		Message:      "message",
		Type:         storage.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Hint:         "hint",
		Domain:       "domain",
		Occurrences:  1,
		CreatedAt:    protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurred: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}
	notificationNew := &storage.Notification{
		Id:           id,
		Level:        storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
		Message:      "message",
		Type:         storage.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Hint:         "hint",
		Domain:       "domain",
		Occurrences:  1,
		CreatedAt:    protoconv.ConvertTimeToTimestamp(time.Unix(100, 0)),
		LastOccurred: protoconv.ConvertTimeToTimestamp(time.Unix(10000, 0)),
	}
	notificationMerged := &storage.Notification{
		Id:           id,
		Level:        storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
		Message:      "message",
		Type:         storage.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Hint:         "hint",
		Domain:       "domain",
		Occurrences:  2,
		CreatedAt:    protoconv.ConvertTimeToTimestamp(time.Unix(100, 0)),
		LastOccurred: protoconv.ConvertTimeToTimestamp(time.Unix(10000, 0)),
	}

	s.store.EXPECT().Get(s.ctx, notificationBase.GetId()).Return(nil, false, nil)
	err := s.writer.Upsert(s.ctx, notificationBase)
	s.Require().NoError(err)

	err = s.writer.Upsert(s.ctx, notificationNew)
	s.Require().NoError(err)

	s.store.EXPECT().UpsertMany(s.ctx, []*storage.Notification{notificationMerged}).Return(nil)
	err = s.writer.Flush(s.ctx)
	s.Require().NoError(err)
}

func (s *writerTestSuite) TestWriteNotification_MergeWithDB() {
	id := "0925514f-3a33-5931-b431-756406e1a008"
	notificationBase := &storage.Notification{
		Id:           id,
		Level:        storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
		Message:      "message",
		Type:         storage.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Hint:         "hint",
		Domain:       "domain",
		Occurrences:  1,
		CreatedAt:    protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurred: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}
	notificationNew := &storage.Notification{
		Id:           id,
		Level:        storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
		Message:      "message",
		Type:         storage.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Hint:         "hint",
		Domain:       "domain",
		Occurrences:  1,
		CreatedAt:    protoconv.ConvertTimeToTimestamp(time.Unix(100, 0)),
		LastOccurred: protoconv.ConvertTimeToTimestamp(time.Unix(10000, 0)),
	}
	notificationMerged := &storage.Notification{
		Id:           id,
		Level:        storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
		Message:      "message",
		Type:         storage.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Hint:         "hint",
		Domain:       "domain",
		Occurrences:  2,
		CreatedAt:    protoconv.ConvertTimeToTimestamp(time.Unix(100, 0)),
		LastOccurred: protoconv.ConvertTimeToTimestamp(time.Unix(10000, 0)),
	}

	s.store.EXPECT().Get(s.ctx, notificationBase.GetId()).Return(notificationBase, true, nil)
	err := s.writer.Upsert(s.ctx, notificationNew)
	s.Require().NoError(err)

	s.store.EXPECT().UpsertMany(s.ctx, []*storage.Notification{notificationMerged}).Return(nil)
	err = s.writer.Flush(s.ctx)
	s.Require().NoError(err)
}

func (s *writerTestSuite) TestWriteNotification_Error() {
	notification := &storage.Notification{
		Level:        storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
		Message:      "message",
		Type:         storage.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Hint:         "hint",
		Domain:       "domain",
		CreatedAt:    protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurred: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}
	enrichedNotification := &storage.Notification{
		Id:           "0925514f-3a33-5931-b431-756406e1a008",
		Level:        storage.NotificationLevel_NOTIFICATION_LEVEL_DANGER,
		Message:      "message",
		Type:         storage.NotificationType_NOTIFICATION_TYPE_GENERIC,
		Hint:         "hint",
		Domain:       "domain",
		Occurrences:  1,
		CreatedAt:    protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurred: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}

	s.store.EXPECT().Get(s.ctx, enrichedNotification.GetId()).Return(nil, false, errFake)
	err := s.writer.Upsert(s.ctx, notification)
	s.ErrorIs(err, errFake)
}
