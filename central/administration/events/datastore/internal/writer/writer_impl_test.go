package writer

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	storeMocks "github.com/stackrox/rox/central/administration/events/datastore/internal/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
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

	s.store.EXPECT().Get(s.ctx, enrichedEvent.GetId()).Return(nil, false, nil)
	err := s.writer.Upsert(s.ctx, event)
	s.Require().NoError(err)

	s.store.EXPECT().UpsertMany(s.ctx, []*storage.AdministrationEvent{enrichedEvent}).Return(nil)
	err = s.writer.Flush(s.ctx)
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

	s.store.EXPECT().Get(s.ctx, eventBase.GetId()).Return(nil, false, nil)
	err := s.writer.Upsert(s.ctx, eventBase)
	s.Require().NoError(err)

	err = s.writer.Upsert(s.ctx, eventNew)
	s.Require().NoError(err)

	s.store.EXPECT().UpsertMany(s.ctx, []*storage.AdministrationEvent{eventMerged}).Return(nil)
	err = s.writer.Flush(s.ctx)
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

	s.store.EXPECT().Get(s.ctx, eventBase.GetId()).Return(eventBase, true, nil)
	err := s.writer.Upsert(s.ctx, eventNew)
	s.Require().NoError(err)

	s.store.EXPECT().UpsertMany(s.ctx, []*storage.AdministrationEvent{eventMerged}).Return(nil)
	err = s.writer.Flush(s.ctx)
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

	s.store.EXPECT().Get(s.ctx, enrichedEvent.GetId()).Return(nil, false, errFake)
	err := s.writer.Upsert(s.ctx, event)
	s.ErrorIs(err, errFake)
}
