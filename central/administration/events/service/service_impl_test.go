package service

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	dsMocks "github.com/stackrox/rox/central/administration/events/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

var (
	errFake   = errors.New("fake error")
	fakeID    = "0925514f-3a33-5931-b431-756406e1a008"
	fakeEvent = &storage.AdministrationEvent{
		Id:             fakeID,
		Level:          storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:        "message",
		Type:           storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:           "hint",
		Domain:         "domain",
		NumOccurrences: 1,
	}
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestAdministrationEventsService(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(countEventsTestSuite))
	suite.Run(t, new(getEventTestSuite))
	suite.Run(t, new(listEventsTestSuite))
}

type baseTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx           context.Context
	datastoreMock *dsMocks.MockDataStore
	service       Service
}

func (s *baseTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = context.Background()
	s.datastoreMock = dsMocks.NewMockDataStore(s.mockCtrl)
	s.service = newService(s.datastoreMock)
}

// Test CountAdministrationEvents

type countEventsTestSuite struct {
	baseTestSuite

	fakeCountEventsRequest  *v1.CountAdministrationEventsRequest
	fakeCountEventsResponse *v1.CountAdministrationEventsResponse
}

func (s *countEventsTestSuite) SetupTest() {
	s.baseTestSuite.SetupTest()

	s.fakeCountEventsRequest = &v1.CountAdministrationEventsRequest{}
	s.fakeCountEventsResponse = &v1.CountAdministrationEventsResponse{Count: 1}
}

func (s *countEventsTestSuite) TestCountAdministrationEvents_Success() {
	s.datastoreMock.EXPECT().CountEvents(s.ctx, gomock.Any()).Return(1, nil)
	result, err := s.service.CountAdministrationEvents(s.ctx, s.fakeCountEventsRequest)

	s.Require().NoError(err)
	s.Equal(s.fakeCountEventsResponse, result)
}

func (s *countEventsTestSuite) TestCountAdministrationEvents_Error() {
	s.datastoreMock.EXPECT().CountEvents(s.ctx, gomock.Any()).Return(0, errFake)
	result, err := s.service.CountAdministrationEvents(s.ctx, s.fakeCountEventsRequest)

	s.ErrorIs(err, errFake)
	s.Nil(result)
}

// Test GetAdministrationEvent

type getEventTestSuite struct {
	baseTestSuite

	fakeResourceByIDRequest *v1.ResourceByID
	fakeGetEventResponse    *v1.GetAdministrationEventResponse
}

func (s *getEventTestSuite) SetupTest() {
	s.baseTestSuite.SetupTest()

	s.fakeResourceByIDRequest = &v1.ResourceByID{Id: fakeID}
	s.fakeGetEventResponse = &v1.GetAdministrationEventResponse{
		Event: toV1Proto(fakeEvent),
	}
}

func (s *getEventTestSuite) TestGetAdministrationEvent_Success() {
	s.datastoreMock.EXPECT().GetEvent(s.ctx, fakeID).Return(fakeEvent, nil)
	result, err := s.service.GetAdministrationEvent(s.ctx, s.fakeResourceByIDRequest)

	s.Require().NoError(err)
	s.Equal(s.fakeGetEventResponse, result)
}

func (s *getEventTestSuite) TestGetAdministrationEvent_Error() {
	s.datastoreMock.EXPECT().GetEvent(s.ctx, fakeID).Return(nil, errFake)
	result, err := s.service.GetAdministrationEvent(s.ctx, s.fakeResourceByIDRequest)

	s.ErrorIs(err, errFake)
	s.Nil(result)
}

func (s *getEventTestSuite) TestGetAdministrationEvent_NotFound() {
	s.datastoreMock.EXPECT().GetEvent(s.ctx, fakeID).Return(nil, errox.NotFound)
	result, err := s.service.GetAdministrationEvent(s.ctx, s.fakeResourceByIDRequest)

	s.ErrorIs(err, errox.NotFound)
	s.Nil(result)
}

// Test ListAdministrationEvents

type listEventsTestSuite struct {
	baseTestSuite

	fakeStorageEventsList  []*storage.AdministrationEvent
	fakeServiceEventsList  []*v1.AdministrationEvent
	fakeListEventsRequest  *v1.ListAdministrationEventsRequest
	fakeListEventsResponse *v1.ListAdministrationEventsResponse
}

func (s *listEventsTestSuite) SetupTest() {
	s.baseTestSuite.SetupTest()

	s.fakeStorageEventsList = []*storage.AdministrationEvent{fakeEvent}
	s.fakeServiceEventsList = []*v1.AdministrationEvent{toV1Proto(fakeEvent)}
	s.fakeListEventsRequest = &v1.ListAdministrationEventsRequest{}
	s.fakeListEventsResponse = &v1.ListAdministrationEventsResponse{Events: s.fakeServiceEventsList}
}

func (s *listEventsTestSuite) TestListAdministrationEvents_Success() {
	s.datastoreMock.EXPECT().ListEvents(s.ctx, gomock.Any()).Return(s.fakeStorageEventsList, nil)
	result, err := s.service.ListAdministrationEvents(s.ctx, s.fakeListEventsRequest)

	s.Require().NoError(err)
	s.Equal(s.fakeListEventsResponse, result)
}

func (s *listEventsTestSuite) TestListAdministrationEvents_Error() {
	s.datastoreMock.EXPECT().ListEvents(s.ctx, gomock.Any()).Return(nil, errFake)
	result, err := s.service.ListAdministrationEvents(s.ctx, s.fakeListEventsRequest)

	s.ErrorIs(err, errFake)
	s.Nil(result)
}

// Test query builder

func TestAdministrationEventsQueryBuilder(t *testing.T) {
	t.Parallel()
	filter := &v1.AdministrationEventsFilter{
		From:         protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		Until:        protoconv.ConvertTimeToTimestamp(time.Unix(10000, 0)),
		Domain:       []string{"domain", "domain"},
		ResourceType: []string{"resourceType", "resourceType"},
		Type:         []v1.AdministrationEventType{v1.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC},
		Level:        []v1.AdministrationEventLevel{v1.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR},
	}
	queryBuilder := getQueryBuilderFromFilter(filter)

	rawQuery, err := queryBuilder.RawQuery()
	require.NoError(t, err)

	assert.Contains(t, rawQuery, "Created Time:tr/1000000-10000000")
	assert.Contains(t, rawQuery, `Event Domain:"domain"`)
	assert.Contains(t, rawQuery, `Event Level:"ADMINISTRATION_EVENT_LEVEL_ERROR"`)
	assert.Contains(t, rawQuery, `Event Type:"ADMINISTRATION_EVENT_TYPE_GENERIC"`)
	assert.Contains(t, rawQuery, `Resource Type:"resourceType"`)
}

func TestAdministrationEventsQueryBuilderNilFilter(t *testing.T) {
	t.Parallel()
	queryBuilder := getQueryBuilderFromFilter(nil)

	rawQuery, err := queryBuilder.RawQuery()
	require.NoError(t, err)

	assert.Empty(t, rawQuery)
}
