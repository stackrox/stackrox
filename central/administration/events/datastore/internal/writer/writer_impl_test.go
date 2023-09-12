package writer

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	storeMocks "github.com/stackrox/rox/central/administration/events/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var errFake = errors.New("fake error")

func TestEventsWriter(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(writerTestSuite))
}

type writerTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	readCtx  context.Context
	writeCtx context.Context
	store    *storeMocks.MockStore
	writer   Writer
}

func (s *writerTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.readCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)
	s.writeCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)
	s.store = storeMocks.NewMockStore(s.mockCtrl)
	s.writer = New(s.store)
}

func (s *writerTestSuite) TestWriteEvent_Success() {
	event := &storage.AdministrationEvent{
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		CreatedAt:      protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}
	enrichedEvent := &storage.AdministrationEvent{
		Id:             "73072ecb-2222-5922-8948-5944338861c8",
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 1,
		CreatedAt:      protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}

	s.store.EXPECT().Get(s.writeCtx, enrichedEvent.GetId()).Return(nil, false, nil)
	err := s.writer.Upsert(s.writeCtx, event)
	s.Require().NoError(err)

	s.store.EXPECT().UpsertMany(s.writeCtx, []*storage.AdministrationEvent{enrichedEvent}).Return(nil)
	err = s.writer.Flush(s.writeCtx)
	s.Require().NoError(err)
}

func (s *writerTestSuite) TestWriteEvent_MergeWithBuffer() {
	id := "73072ecb-2222-5922-8948-5944338861c8"
	eventBase := &storage.AdministrationEvent{
		Id:             id,
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 1,
		CreatedAt:      protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}
	eventNew := &storage.AdministrationEvent{
		Id:             id,
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 1,
		CreatedAt:      protoconv.ConvertTimeToTimestamp(time.Unix(100, 0)),
		LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Unix(10000, 0)),
	}
	eventMerged := &storage.AdministrationEvent{
		Id:             id,
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 2,
		CreatedAt:      protoconv.ConvertTimeToTimestamp(time.Unix(100, 0)),
		LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Unix(10000, 0)),
	}

	s.store.EXPECT().Get(s.writeCtx, eventBase.GetId()).Return(nil, false, nil)
	err := s.writer.Upsert(s.writeCtx, eventBase)
	s.Require().NoError(err)

	err = s.writer.Upsert(s.writeCtx, eventNew)
	s.Require().NoError(err)

	s.store.EXPECT().UpsertMany(s.writeCtx, []*storage.AdministrationEvent{eventMerged}).Return(nil)
	err = s.writer.Flush(s.writeCtx)
	s.Require().NoError(err)
}

func (s *writerTestSuite) TestWriteEvent_MergeWithDB() {
	id := "73072ecb-2222-5922-8948-5944338861c8"
	eventBase := &storage.AdministrationEvent{
		Id:             id,
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 1,
		CreatedAt:      protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}
	eventNew := &storage.AdministrationEvent{
		Id:             id,
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 1,
		CreatedAt:      protoconv.ConvertTimeToTimestamp(time.Unix(100, 0)),
		LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Unix(10000, 0)),
	}
	eventMerged := &storage.AdministrationEvent{
		Id:             id,
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 2,
		CreatedAt:      protoconv.ConvertTimeToTimestamp(time.Unix(100, 0)),
		LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Unix(10000, 0)),
	}

	s.store.EXPECT().Get(s.writeCtx, eventBase.GetId()).Return(eventBase, true, nil)
	err := s.writer.Upsert(s.writeCtx, eventNew)
	s.Require().NoError(err)

	s.store.EXPECT().UpsertMany(s.writeCtx, []*storage.AdministrationEvent{eventMerged}).Return(nil)
	err = s.writer.Flush(s.writeCtx)
	s.Require().NoError(err)
}

func (s *writerTestSuite) TestWriteEvent_Error() {
	event := &storage.AdministrationEvent{
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		CreatedAt:      protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}
	enrichedEvent := &storage.AdministrationEvent{
		Id:             "73072ecb-2222-5922-8948-5944338861c8",
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 1,
		CreatedAt:      protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}

	s.store.EXPECT().Get(s.writeCtx, enrichedEvent.GetId()).Return(nil, false, errFake)
	err := s.writer.Upsert(s.writeCtx, event)
	s.ErrorIs(err, errFake)
}

func (s *writerTestSuite) TestWriteEvent_WriteBufferExhaustedIsRetryable() {
	event := &storage.AdministrationEvent{
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		CreatedAt:      protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}

	maxWriterSize = 0
	defer func() { maxWriterSize = 1000 }()
	err := s.writer.Upsert(s.writeCtx, event)
	s.Require().Equal(err.Error(), errWriteBufferExhausted.Error())
	s.True(retry.IsRetryable(err))
}

func (s *writerTestSuite) TestWriteEvent_SACNoWrite_Error() {
	event := &storage.AdministrationEvent{
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		CreatedAt:      protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
	}

	err := s.writer.Upsert(s.readCtx, event)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
}

func (s *writerTestSuite) TestFlushEvents_SACNoWrite_Error() {
	err := s.writer.Flush(s.readCtx)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
}
