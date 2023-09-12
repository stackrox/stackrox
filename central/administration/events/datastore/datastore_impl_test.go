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
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	errFake   = errors.New("fake error")
	fakeQuery = &v1.Query{}
)

func TestEventsDatastore(t *testing.T) {
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
	s.datastore = NewDataStore(s.searcher, s.store, s.writer)
}

func (s *datastoreTestSuite) TestAddEvent_Success() {
	event := &events.AdministrationEvent{
		Domain:  "domain",
		Hint:    "hint",
		Level:   storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message: "message",
		Type:    storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
	}

	s.writer.EXPECT().Upsert(s.ctx, event).Return(nil)
	err := s.datastore.AddEvent(s.ctx, event)

	s.NoError(err)
}

func (s *datastoreTestSuite) TestAddEvent_Error() {
	event := &events.AdministrationEvent{
		Domain:  "domain",
		Hint:    "hint",
		Level:   storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message: "message",
		Type:    storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
	}

	s.writer.EXPECT().Upsert(s.ctx, event).Return(errFake)
	err := s.datastore.AddEvent(s.ctx, event)

	s.ErrorIs(err, errFake)
}

func (s *datastoreTestSuite) TestCountEvents_Success() {
	count := 10
	s.searcher.EXPECT().Count(s.ctx, fakeQuery).Return(count, nil)

	result, err := s.datastore.CountEvents(s.ctx, fakeQuery)

	s.Require().NoError(err)
	s.Equal(count, result)
}

func (s *datastoreTestSuite) TestCountEvents_Error() {
	s.searcher.EXPECT().Count(s.ctx, fakeQuery).Return(0, errFake)

	_, err := s.datastore.CountEvents(s.ctx, fakeQuery)

	s.ErrorIs(err, errFake)
}

func (s *datastoreTestSuite) TestGetEvent_Success() {
	id := "0925514f-3a33-5931-b431-756406e1a008"
	event := &storage.AdministrationEvent{
		Id:             id,
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 1,
	}

	s.store.EXPECT().Get(s.ctx, id).Return(event, true, nil)
	result, err := s.datastore.GetEvent(s.ctx, id)

	s.Require().NoError(err)
	s.Equal(event, result)
}

func (s *datastoreTestSuite) TestGetEvent_Error() {
	id := "0925514f-3a33-5931-b431-756406e1a008"

	s.store.EXPECT().Get(s.ctx, id).Return(nil, false, errFake)
	_, err := s.datastore.GetEvent(s.ctx, id)

	s.ErrorIs(err, errFake)
}

func (s *datastoreTestSuite) TestGetEvent_NotFound() {
	id := "0925514f-3a33-5931-b431-756406e1a008"

	s.store.EXPECT().Get(s.ctx, id).Return(nil, false, nil)
	_, err := s.datastore.GetEvent(s.ctx, id)

	s.ErrorIs(err, errox.NotFound)
}

func (s *datastoreTestSuite) TestListEvents_Success() {
	events := []*storage.AdministrationEvent{
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

	s.store.EXPECT().GetByQuery(s.ctx, fakeQuery).Return(events, nil)
	result, err := s.datastore.ListEvents(s.ctx, fakeQuery)

	s.Require().NoError(err)
	s.Equal(events, result)
}

func (s *datastoreTestSuite) TestListEvents_Error() {
	s.store.EXPECT().GetByQuery(s.ctx, fakeQuery).Return(nil, errFake)

	_, err := s.datastore.ListEvents(s.ctx, fakeQuery)

	s.ErrorIs(err, errFake)
}
