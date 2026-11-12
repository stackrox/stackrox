package service

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	dsMocks "github.com/stackrox/rox/central/administration/events/datastore/mocks"
	eventViewMocks "github.com/stackrox/rox/central/views/administrationevents/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/search"
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
	suite.Run(t, new(countEventsTestSuite))
	suite.Run(t, new(getEventTestSuite))
	suite.Run(t, new(listEventsTestSuite))
}

type baseTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx           context.Context
	datastoreMock *dsMocks.MockDataStore
	viewMock      *eventViewMocks.MockEventView
	service       Service
}

func (s *baseTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = context.Background()
	s.datastoreMock = dsMocks.NewMockDataStore(s.mockCtrl)
	s.viewMock = eventViewMocks.NewMockEventView(s.mockCtrl)
	s.service = newService(s.datastoreMock, s.viewMock)
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
	s.viewMock.EXPECT().Count(s.ctx, gomock.Any()).Return(1, nil)
	result, err := s.service.CountAdministrationEvents(s.ctx, s.fakeCountEventsRequest)

	s.Require().NoError(err)
	protoassert.Equal(s.T(), s.fakeCountEventsResponse, result)
}

func (s *countEventsTestSuite) TestCountAdministrationEvents_Error() {
	s.viewMock.EXPECT().Count(s.ctx, gomock.Any()).Return(0, errFake)
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
	protoassert.Equal(s.T(), s.fakeGetEventResponse, result)
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
	s.viewMock.EXPECT().Get(s.ctx, gomock.Any()).Return(s.fakeStorageEventsList, nil)
	result, err := s.service.ListAdministrationEvents(s.ctx, s.fakeListEventsRequest)

	s.Require().NoError(err)
	protoassert.Equal(s.T(), s.fakeListEventsResponse, result)
}

func (s *listEventsTestSuite) TestListAdministrationEvents_Error() {
	s.viewMock.EXPECT().Get(s.ctx, gomock.Any()).Return(nil, errFake)
	result, err := s.service.ListAdministrationEvents(s.ctx, s.fakeListEventsRequest)

	s.ErrorIs(err, errFake)
	s.Nil(result)
}

// Test query builder

func TestAdministrationEventsQueryBuilder(t *testing.T) {
	filter := &v1.AdministrationEventsFilter{
		From:         protoconv.ConvertTimeToTimestamp(time.Unix(1000, 0)),
		Until:        protoconv.ConvertTimeToTimestamp(time.Unix(10000, 0)),
		Domain:       []string{"domain", "domain"},
		ResourceType: []string{"resourceType", "resourceType"},
		Type:         []v1.AdministrationEventType{v1.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC},
		Level:        []v1.AdministrationEventLevel{v1.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR},
	}
	query, err := getQueryFromFilter(filter)
	require.NoError(t, err)

	queryStr := query.String()
	assert.Contains(t, queryStr, `Resource Type`)
	assert.Contains(t, queryStr, `Event Domain`)
	assert.Contains(t, queryStr, `Event Level`)
	assert.Contains(t, queryStr, `Event Type`)
	assert.Contains(t, queryStr, `Created Time`)
}

func TestWorkloadFilterSkippedForNonImageResourceType(t *testing.T) {
	filter := &v1.AdministrationEventsFilter{
		ResourceType:  []string{"Node"},
		WorkloadQuery: `Cluster:"prod"`,
	}
	query, err := getQueryFromFilter(filter)
	require.NoError(t, err)

	queryStr := query.String()
	assert.Contains(t, queryStr, "Node")
	assert.NotContains(t, queryStr, "Resource ID")
}

func TestAdministrationEventsQueryBuilderNilFilter(t *testing.T) {
	query, err := getQueryFromFilter(nil)
	require.NoError(t, err)
	assert.NotNil(t, query)
}

func TestWorkloadQuery(t *testing.T) {
	for name, tc := range map[string]struct {
		resourceTypes []string
		workloadQuery string
		applies       bool
	}{
		"all types":         {workloadQuery: `Cluster:"prod"`, applies: true},
		"image only":        {resourceTypes: []string{"Image"}, workloadQuery: `Cluster:"prod"`, applies: true},
		"mixed types":       {resourceTypes: []string{"Image", "Node"}, workloadQuery: `Cluster:"prod"`, applies: true},
		"other types":       {resourceTypes: []string{"Node"}, workloadQuery: `Cluster:"prod"`},
		"no workload query": {},
	} {
		t.Run(name, func(t *testing.T) {
			query, err := getQueryFromFilter(&v1.AdministrationEventsFilter{ResourceType: tc.resourceTypes, WorkloadQuery: tc.workloadQuery})
			require.NoError(t, err)
			var workload *v1.MatchFieldSubQuery
			search.ApplyFnToAllBaseQueries(query, func(base *v1.BaseQuery) {
				if sub := base.GetMatchFieldSubQuery(); sub != nil {
					workload = sub
				}
			})
			if tc.applies {
				expected, err := search.ParseQuery(tc.workloadQuery)
				require.NoError(t, err)
				expected.Selects = []*v1.QuerySelect{search.NewQuerySelect(search.ImageID).Proto()}
				require.NotNil(t, workload)
				assert.Equal(t, search.ResourceID.String(), workload.GetField())
				assert.Equal(t, v1.SearchCategory_DEPLOYMENTS, workload.GetCategory())
				protoassert.Equal(t, expected, workload.GetQuery())
			} else {
				assert.Nil(t, workload)
			}
		})
	}
}

func TestWorkloadQueryErrors(t *testing.T) {
	for name, rawQuery := range map[string]string{
		"invalid query": "INVALIDQUERY",
		"view error":    "Cluster:prod",
	} {
		for _, rpc := range []string{"count", "list"} {
			t.Run(name+"/"+rpc, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				ds := dsMocks.NewMockDataStore(ctrl)
				view := eventViewMocks.NewMockEventView(ctrl)
				expectedError := error(errox.InvalidArgs)
				if name == "view error" {
					if rpc == "count" {
						view.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, errFake)
					} else {
						view.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, errFake)
					}
					expectedError = errFake
				}
				svc := newService(ds, view)
				filter := &v1.AdministrationEventsFilter{WorkloadQuery: rawQuery}
				var err error
				if rpc == "count" {
					_, err = svc.CountAdministrationEvents(context.Background(), &v1.CountAdministrationEventsRequest{Filter: filter})
				} else {
					_, err = svc.ListAdministrationEvents(context.Background(), &v1.ListAdministrationEventsRequest{Filter: filter})
				}
				assert.ErrorIs(t, err, expectedError)
			})
		}
	}
}
