package datastore

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	searchMocks "github.com/stackrox/rox/central/administration/events/datastore/internal/search/mocks"
	storeMocks "github.com/stackrox/rox/central/administration/events/datastore/internal/store/mocks"
	writerMocks "github.com/stackrox/rox/central/administration/events/datastore/internal/writer/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	errFake   = errors.New("fake error")
	fakeQuery = &v1.Query{}
)

func TestNotificationsDatastore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(datastoreTestSuite))
}

type datastoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx       context.Context
	datastore DataStore
	searcher  *searchMocks.MockSearcher
	store     *storeMocks.MockStore
	writer    *writerMocks.MockWriter
}

func (s *datastoreTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.ctx = context.Background()
	s.searcher = searchMocks.NewMockSearcher(s.mockCtrl)
	s.store = storeMocks.NewMockStore(s.mockCtrl)
	s.writer = writerMocks.NewMockWriter(s.mockCtrl)
	s.datastore = newDataStore(s.searcher, s.store, s.writer)
}

func (s *datastoreTestSuite) TestAddNotification_Success() {
	notification := &storage.AdministrationEvent{
		Id:             "0925514f-3a33-5931-b431-756406e1a008",
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 1,
	}

	s.writer.EXPECT().Upsert(s.ctx, notification).Return(nil)
	err := s.datastore.AddEvent(s.ctx, notification)

	s.NoError(err)
}

func (s *datastoreTestSuite) TestAddNotification_Error() {
	notification := &storage.AdministrationEvent{
		Id:             "0925514f-3a33-5931-b431-756406e1a008",
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 1,
	}

	s.writer.EXPECT().Upsert(s.ctx, notification).Return(errFake)
	err := s.datastore.AddEvent(s.ctx, notification)

	s.ErrorIs(err, errFake)
}

func (s *datastoreTestSuite) TestCountNotifications_Success() {
	count := 10
	s.searcher.EXPECT().Count(s.ctx, fakeQuery).Return(count, nil)

	result, err := s.datastore.CountEvents(s.ctx, fakeQuery)

	s.Require().NoError(err)
	s.Equal(count, result)
}

func (s *datastoreTestSuite) TestCountNotifications_Error() {
	s.searcher.EXPECT().Count(s.ctx, fakeQuery).Return(0, errFake)

	_, err := s.datastore.CountEvents(s.ctx, fakeQuery)

	s.ErrorIs(err, errFake)
}

func (s *datastoreTestSuite) TestGetNotification_Success() {
	id := "0925514f-3a33-5931-b431-756406e1a008"
	notification := &storage.AdministrationEvent{
		Id:             id,
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 1,
	}

	s.store.EXPECT().Get(s.ctx, id).Return(notification, true, nil)
	result, err := s.datastore.GetEventByID(s.ctx, id)

	s.Require().NoError(err)
	s.Equal(notification, result)
}

func (s *datastoreTestSuite) TestGetNotification_Error() {
	id := "0925514f-3a33-5931-b431-756406e1a008"

	s.store.EXPECT().Get(s.ctx, id).Return(nil, false, errFake)
	_, err := s.datastore.GetEventByID(s.ctx, id)

	s.ErrorIs(err, errFake)
}

func (s *datastoreTestSuite) TestGetNotification_NotFound() {
	id := "0925514f-3a33-5931-b431-756406e1a008"

	s.store.EXPECT().Get(s.ctx, id).Return(nil, false, nil)
	_, err := s.datastore.GetEventByID(s.ctx, id)

	s.ErrorIs(err, errox.NotFound)
}

func (s *datastoreTestSuite) TestListNotifications_Success() {
	notifications := []*storage.AdministrationEvent{
		{
			Id:             "0925514f-3a33-5931-b431-756406e1a008",
			Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
			Message:        "message",
			Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
			Hint:           "hint",
			Domain:         "domain",
			NumOccurrences: 1,
		},
	}

	s.store.EXPECT().GetByQuery(s.ctx, fakeQuery).Return(notifications, nil)
	result, err := s.datastore.ListEvents(s.ctx, fakeQuery)

	s.Require().NoError(err)
	s.Equal(notifications, result)
}

func (s *datastoreTestSuite) TestListNotifications_Error() {
	s.store.EXPECT().GetByQuery(s.ctx, fakeQuery).Return(nil, errFake)

	_, err := s.datastore.ListEvents(s.ctx, fakeQuery)

	s.ErrorIs(err, errFake)
}
