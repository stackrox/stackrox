package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/types"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/alert/datastore"
	indexMocks "github.com/stackrox/rox/central/alert/index/mocks"
	searchMocks "github.com/stackrox/rox/central/alert/search/mocks"
	storeMocks "github.com/stackrox/rox/central/alert/store/mocks"
	"github.com/stackrox/rox/central/alerttest"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	fakeQuery = "fakeQuery"
)

var (
	errFake = errors.New("fake error")
)

func TestAlertService(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(getAlertTests))
	suite.Run(t, new(listAlertsTests))
	suite.Run(t, new(getAlertsGroupsTests))
	suite.Run(t, new(getAlertsCountsTests))
	suite.Run(t, new(getAlertTimeseriesTests))
}

type getAlertTests struct {
	suite.Suite

	storage  *storeMocks.Store
	indexer  *indexMocks.Indexer
	searcher *searchMocks.Searcher
	service  Service

	fakeResourceByIDRequest *v1.ResourceByID
}

func (s *getAlertTests) SetupTest() {
	s.storage = new(storeMocks.Store)
	s.indexer = new(indexMocks.Indexer)
	s.searcher = new(searchMocks.Searcher)

	dataStore := datastore.New(s.storage, s.indexer, s.searcher)

	s.service = New(dataStore)

	s.fakeResourceByIDRequest = &v1.ResourceByID{
		Id: alerttest.FakeAlertID,
	}
}

func (s *getAlertTests) TestGetAlert() {
	fakeAlert := alerttest.NewFakeAlert()

	s.storage.On("GetAlert", alerttest.FakeAlertID).Return(fakeAlert, true, nil)

	result, err := s.service.GetAlert(context.Background(), s.fakeResourceByIDRequest)

	s.NoError(err)
	s.Equal(fakeAlert, result)
}

func (s *getAlertTests) TestGetAlertWhenTheDataAccessLayerFails() {
	s.storage.On("GetAlert", alerttest.FakeAlertID).Return(alerttest.NewFakeAlert(), false, errFake)

	result, err := s.service.GetAlert(context.Background(), s.fakeResourceByIDRequest)

	s.Equal(status.Error(codes.Internal, "fake error"), err)
	s.Equal((*v1.Alert)(nil), result)
}

func (s *getAlertTests) TestGetAlertWhenAlertIsMissing() {
	s.storage.On("GetAlert", alerttest.FakeAlertID).Return(nil, false, nil)

	result, err := s.service.GetAlert(context.Background(), s.fakeResourceByIDRequest)

	s.Equal(status.Errorf(codes.NotFound, "alert with id '%s' does not exist", alerttest.FakeAlertID), err)
	s.Equal((*v1.Alert)(nil), result)
}

type listAlertsTests struct {
	suite.Suite

	storage  *storeMocks.Store
	indexer  *indexMocks.Indexer
	searcher *searchMocks.Searcher
	service  Service

	fakeListAlertSlice         []*v1.ListAlert
	expectedListAlertsResponse *v1.ListAlertsResponse
}

func (s *listAlertsTests) SetupTest() {
	s.storage = new(storeMocks.Store)
	s.indexer = new(indexMocks.Indexer)
	s.searcher = new(searchMocks.Searcher)

	dataStore := datastore.New(s.storage, s.indexer, s.searcher)

	s.service = New(dataStore)

	s.fakeListAlertSlice = []*v1.ListAlert{
		{
			Id: "id1",
			Time: &types.Timestamp{
				Seconds: 1,
				Nanos:   2,
			},
			Policy: &v1.ListAlertPolicy{
				Id: alerttest.FakePolicyID,
			},
		},
		{
			Id: "id2",
			Time: &types.Timestamp{
				Seconds: 1,
				Nanos:   1,
			},
			Policy: &v1.ListAlertPolicy{
				Id: alerttest.FakePolicyID,
			},
		},
		{
			Id: "id3",
			Time: &types.Timestamp{
				Seconds: 2,
				Nanos:   0,
			},
			Policy: &v1.ListAlertPolicy{
				Id: alerttest.FakePolicyID,
			},
		},
		{
			Id: "id4",
			Time: &types.Timestamp{
				Seconds: 2,
				Nanos:   0,
			},
			Policy: &v1.ListAlertPolicy{
				Id: alerttest.FakePolicyID,
			},
		},
	}

	// expectedListAlertsResponse includes the slice of ListAlert objects sorted stably by descending timestamp
	s.expectedListAlertsResponse = &v1.ListAlertsResponse{
		Alerts: []*v1.ListAlert{
			{
				Id: "id3",
				Time: &types.Timestamp{
					Seconds: 2,
					Nanos:   0,
				},
				Policy: &v1.ListAlertPolicy{
					Id: alerttest.FakePolicyID,
				},
			},
			{
				Id: "id4",
				Time: &types.Timestamp{
					Seconds: 2,
					Nanos:   0,
				},
				Policy: &v1.ListAlertPolicy{
					Id: alerttest.FakePolicyID,
				},
			},
			{
				Id: "id1",
				Time: &types.Timestamp{
					Seconds: 1,
					Nanos:   2,
				},
				Policy: &v1.ListAlertPolicy{
					Id: alerttest.FakePolicyID,
				},
			},
			{
				Id: "id2",
				Time: &types.Timestamp{
					Seconds: 1,
					Nanos:   1,
				},
				Policy: &v1.ListAlertPolicy{
					Id: alerttest.FakePolicyID,
				},
			},
		},
	}
}

func (s *listAlertsTests) TestListAlerts() {
	fakeQuery := search.NewQueryBuilder().AddStrings(search.DeploymentName, "field1", "field12").AddStrings(search.Category, "field2")

	s.searcher.On("SearchListAlerts", fakeQuery.ToParsedSearchRequest()).Return(s.fakeListAlertSlice, nil)

	result, err := s.service.ListAlerts(context.Background(), &v1.ListAlertsRequest{
		Query: fakeQuery.Query(),
	})

	s.NoError(err)
	s.Equal(s.expectedListAlertsResponse, result)
}

func (s *listAlertsTests) TestListAlertsWhenTheQueryIsEmpty() {
	s.searcher.On("SearchListAlerts", &v1.ParsedSearchRequest{}).Return(s.fakeListAlertSlice, nil)

	result, err := s.service.ListAlerts(context.Background(), &v1.ListAlertsRequest{
		Query: "",
	})

	s.NoError(err)
	s.Equal(s.expectedListAlertsResponse, result)
}

func (s *listAlertsTests) TestListAlertsWhenTheQueryIsInvalid() {
	result, err := s.service.ListAlerts(context.Background(), &v1.ListAlertsRequest{
		Query: fakeQuery,
	})

	s.Equal(err, status.Error(codes.Internal, "After parsing, query is empty"))
	s.Equal((*v1.ListAlertsResponse)(nil), result)
}

func (s *listAlertsTests) TestListAlertsWhenTheDataLayerFails() {
	s.searcher.On("SearchListAlerts", &v1.ParsedSearchRequest{}).Return(nil, errFake)

	result, err := s.service.ListAlerts(context.Background(), &v1.ListAlertsRequest{
		Query: "",
	})

	s.Equal(status.Error(codes.Internal, "fake error"), err)
	s.Equal((*v1.ListAlertsResponse)(nil), result)
}

type getAlertsGroupsTests struct {
	suite.Suite

	storage  *storeMocks.Store
	indexer  *indexMocks.Indexer
	searcher *searchMocks.Searcher
	service  Service
}

func (s *getAlertsGroupsTests) SetupTest() {
	s.storage = new(storeMocks.Store)
	s.indexer = new(indexMocks.Indexer)
	s.searcher = new(searchMocks.Searcher)

	dataStore := datastore.New(s.storage, s.indexer, s.searcher)

	s.service = New(dataStore)
}

func (s *getAlertsGroupsTests) TestGetAlertsGroupForOneCategory() {
	fakeListAlertSlice := []*v1.ListAlert{
		{
			Id: "id1",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Id:         "id1",
				Name:       "policy1",
				Severity:   v1.Severity_LOW_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 300},
		},
		{
			Id: "id2",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Id:         "id2",
				Name:       "policy2",
				Severity:   v1.Severity_HIGH_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 200},
		},
		{
			Id: "id3",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Id:         "id1",
				Name:       "policy1",
				Severity:   v1.Severity_LOW_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 100},
		},
	}

	expected := &v1.GetAlertsGroupResponse{
		AlertsByPolicies: []*v1.GetAlertsGroupResponse_PolicyGroup{
			{
				Policy: &v1.ListAlertPolicy{
					Categories: []string{"Image Assurance"},
					Id:         "id1",
					Name:       "policy1",
					Severity:   v1.Severity_LOW_SEVERITY,
				},
				NumAlerts: 2,
			},
			{
				Policy: &v1.ListAlertPolicy{
					Categories: []string{"Image Assurance"},
					Id:         "id2",
					Name:       "policy2",
					Severity:   v1.Severity_HIGH_SEVERITY,
				},
				NumAlerts: 1,
			},
		},
	}

	s.testGetAlertsGroupFor(fakeListAlertSlice, expected)
}

func (s *getAlertsGroupsTests) TestGetAlertsGroupForMultipleCategories() {
	fakeListAlertSlice := []*v1.ListAlert{
		{
			Id: "id1",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Id:         "id1",
				Name:       "policy1",
				Severity:   v1.Severity_LOW_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 300},
		},
		{
			Id: "id2",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance", "Privileges Capabilities"},
				Id:         "id2",
				Name:       "policy2",
				Severity:   v1.Severity_HIGH_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 200},
		},
		{
			Id: "id3",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Container Configuration"},
				Id:         "id30",
				Name:       "policy30",
				Severity:   v1.Severity_CRITICAL_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 150},
		},
		{
			Id: "id4",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Id:         "id1",
				Name:       "policy1",
				Severity:   v1.Severity_LOW_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 100},
		},
	}

	expected := &v1.GetAlertsGroupResponse{
		AlertsByPolicies: []*v1.GetAlertsGroupResponse_PolicyGroup{
			{
				Policy: &v1.ListAlertPolicy{
					Categories: []string{"Image Assurance"},
					Id:         "id1",
					Name:       "policy1",
					Severity:   v1.Severity_LOW_SEVERITY,
				},
				NumAlerts: 2,
			},
			{
				Policy: &v1.ListAlertPolicy{
					Categories: []string{"Image Assurance", "Privileges Capabilities"},
					Id:         "id2",
					Name:       "policy2",
					Severity:   v1.Severity_HIGH_SEVERITY,
				},
				NumAlerts: 1,
			},
			{
				Policy: &v1.ListAlertPolicy{
					Categories: []string{"Container Configuration"},
					Id:         "id30",
					Name:       "policy30",
					Severity:   v1.Severity_CRITICAL_SEVERITY,
				},
				NumAlerts: 1,
			},
		},
	}

	s.testGetAlertsGroupFor(fakeListAlertSlice, expected)
}

func (s *getAlertsGroupsTests) testGetAlertsGroupFor(fakeListAlertSlice []*v1.ListAlert, expected *v1.GetAlertsGroupResponse) {
	s.searcher.On("SearchListAlerts", &v1.ParsedSearchRequest{}).Return(fakeListAlertSlice, nil)

	result, err := s.service.GetAlertsGroup(context.Background(), &v1.ListAlertsRequest{
		Query: "",
	})

	s.NoError(err)
	s.Equal(expected, result)
}

func (s *getAlertsGroupsTests) TestGetAlertsGroupWhenTheDataAccessLayerFails() {
	s.searcher.On("SearchListAlerts", &v1.ParsedSearchRequest{}).Return(nil, errFake)

	result, err := s.service.GetAlertsGroup(context.Background(), &v1.ListAlertsRequest{
		Query: "",
	})

	s.Equal(status.Error(codes.Internal, "fake error"), err)
	s.Equal((*v1.GetAlertsGroupResponse)(nil), result)
}

type getAlertsCountsTests struct {
	suite.Suite

	storage  *storeMocks.Store
	indexer  *indexMocks.Indexer
	searcher *searchMocks.Searcher
	service  Service
}

func (s *getAlertsCountsTests) SetupTest() {
	s.storage = new(storeMocks.Store)
	s.indexer = new(indexMocks.Indexer)
	s.searcher = new(searchMocks.Searcher)

	dataStore := datastore.New(s.storage, s.indexer, s.searcher)

	s.service = New(dataStore)
}

func (s *getAlertsCountsTests) TestGetAlertsCountsWhenAlertsAreNotGrouped() {
	fakeListAlertSlice := []*v1.ListAlert{
		{
			Id: "id1",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Name:       "policy1",
				Severity:   v1.Severity_LOW_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 300},
		},
		{
			Id: "id2",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Container Configuration"},
				Name:       "policy2",
				Severity:   v1.Severity_CRITICAL_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 200},
		},
		{
			Id: "id3",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Name:       "policy1",
				Severity:   v1.Severity_LOW_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 130},
		},
		{
			Id: "id4",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Privileges Capabilities"},
				Name:       "policy3",
				Severity:   v1.Severity_MEDIUM_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 120},
		},
		{
			Id: "id5",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance", "Container Configuration"},
				Name:       "policy4",
				Severity:   v1.Severity_HIGH_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 120},
		},
		{
			Id: "id6",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance", "Container Configuration"},
				Name:       "policy4",
				Severity:   v1.Severity_HIGH_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 110},
		},
	}

	expected := &v1.GetAlertsCountsResponse{
		Groups: []*v1.GetAlertsCountsResponse_AlertGroup{
			{
				Group: "",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					{
						Severity: v1.Severity_LOW_SEVERITY,
						Count:    2,
					},
					{
						Severity: v1.Severity_MEDIUM_SEVERITY,
						Count:    1,
					},
					{
						Severity: v1.Severity_HIGH_SEVERITY,
						Count:    2,
					},
					{
						Severity: v1.Severity_CRITICAL_SEVERITY,
						Count:    1,
					},
				},
			},
		},
	}

	s.testGetAlertCounts(fakeListAlertSlice, v1.GetAlertsCountsRequest_UNSET, expected)
}

func (s *getAlertsCountsTests) TestGetAlertsCountsForAlertsGroupedByCategory() {
	fakeListAlertSlice := []*v1.ListAlert{
		{
			Id: "id1",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Name:       "policy1",
				Severity:   v1.Severity_LOW_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 300},
		},
		{
			Id: "id2",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Container Configuration"},
				Name:       "policy2",
				Severity:   v1.Severity_CRITICAL_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 200},
		},
		{
			Id: "id3",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Name:       "policy1",
				Severity:   v1.Severity_LOW_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 130},
		},
		{
			Id: "id4",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Privileges Capabilities"},
				Name:       "policy3",
				Severity:   v1.Severity_MEDIUM_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 120},
		},
		{
			Id: "id5",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance", "Container Configuration"},
				Name:       "policy4",
				Severity:   v1.Severity_HIGH_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 120},
		},
		{
			Id: "id6",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance", "Container Configuration"},
				Name:       "policy4",
				Severity:   v1.Severity_HIGH_SEVERITY,
			},
			Time: &timestamp.Timestamp{Seconds: 110},
		},
	}

	expected := &v1.GetAlertsCountsResponse{
		Groups: []*v1.GetAlertsCountsResponse_AlertGroup{
			{
				Group: "Container Configuration",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					{
						Severity: v1.Severity_HIGH_SEVERITY,
						Count:    2,
					},
					{
						Severity: v1.Severity_CRITICAL_SEVERITY,
						Count:    1,
					},
				},
			},
			{
				Group: "Image Assurance",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					{
						Severity: v1.Severity_LOW_SEVERITY,
						Count:    2,
					},
					{
						Severity: v1.Severity_HIGH_SEVERITY,
						Count:    2,
					},
				},
			},
			{
				Group: "Privileges Capabilities",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					{
						Severity: v1.Severity_MEDIUM_SEVERITY,
						Count:    1,
					},
				},
			},
		},
	}

	s.testGetAlertCounts(fakeListAlertSlice, v1.GetAlertsCountsRequest_CATEGORY, expected)
}

func (s *getAlertsCountsTests) TestGetAlertsCountsForAlertsGroupedByCluster() {
	fakeListAlertSlice := []*v1.ListAlert{
		{
			Id: "id1",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Name:       "policy1",
				Severity:   v1.Severity_LOW_SEVERITY,
			},
			Deployment: &v1.ListAlertDeployment{
				ClusterName: "test",
			},
			Time: &timestamp.Timestamp{Seconds: 300},
		},
		{
			Id: "id2",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Container Configuration"},
				Name:       "policy2",
				Severity:   v1.Severity_CRITICAL_SEVERITY,
			},
			Deployment: &v1.ListAlertDeployment{
				ClusterName: "test",
			},
			Time: &timestamp.Timestamp{Seconds: 200},
		},
		{
			Id: "id3",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Name:       "policy1",
				Severity:   v1.Severity_LOW_SEVERITY,
			},
			Deployment: &v1.ListAlertDeployment{
				ClusterName: "prod",
			},
			Time: &timestamp.Timestamp{Seconds: 130},
		},
		{
			Id: "id4",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Privileges Capabilities"},
				Name:       "policy3",
				Severity:   v1.Severity_MEDIUM_SEVERITY,
			},
			Deployment: &v1.ListAlertDeployment{
				ClusterName: "prod",
			},
			Time: &timestamp.Timestamp{Seconds: 120},
		},
		{
			Id: "id5",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance", "Container Configuration"},
				Name:       "policy4",
				Severity:   v1.Severity_HIGH_SEVERITY,
			},
			Deployment: &v1.ListAlertDeployment{
				ClusterName: "prod",
			},
			Time: &timestamp.Timestamp{Seconds: 120},
		},
		{
			Id: "id6",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance", "Container Configuration"},
				Name:       "policy4",
				Severity:   v1.Severity_HIGH_SEVERITY,
			},
			Deployment: &v1.ListAlertDeployment{
				ClusterName: "test",
			},
			Time: &timestamp.Timestamp{Seconds: 110},
		},
	}

	expected := &v1.GetAlertsCountsResponse{
		Groups: []*v1.GetAlertsCountsResponse_AlertGroup{
			{
				Group: "prod",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					{
						Severity: v1.Severity_LOW_SEVERITY,
						Count:    1,
					},
					{
						Severity: v1.Severity_MEDIUM_SEVERITY,
						Count:    1,
					},
					{
						Severity: v1.Severity_HIGH_SEVERITY,
						Count:    1,
					},
				},
			},
			{
				Group: "test",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					{
						Severity: v1.Severity_LOW_SEVERITY,
						Count:    1,
					},
					{
						Severity: v1.Severity_HIGH_SEVERITY,
						Count:    1,
					},
					{
						Severity: v1.Severity_CRITICAL_SEVERITY,
						Count:    1,
					},
				},
			},
		},
	}

	s.testGetAlertCounts(fakeListAlertSlice, v1.GetAlertsCountsRequest_CLUSTER, expected)
}

func (s *getAlertsCountsTests) testGetAlertCounts(fakeListAlertSlice []*v1.ListAlert, groupBy v1.GetAlertsCountsRequest_RequestGroup, expected *v1.GetAlertsCountsResponse) {
	s.searcher.On("SearchListAlerts", &v1.ParsedSearchRequest{}).Return(fakeListAlertSlice, nil)

	result, err := s.service.GetAlertsCounts(context.Background(), &v1.GetAlertsCountsRequest{Request: &v1.ListAlertsRequest{
		Query: "",
	}, GroupBy: groupBy})

	s.NoError(err)
	s.Equal(expected, result)
}

func (s *getAlertsCountsTests) TestGetAlertsCountsWhenTheGroupIsUnknown() {
	const unknownGroupBy = v1.GetAlertsCountsRequest_RequestGroup(-99)

	fakeListAlertSlice := []*v1.ListAlert{
		{
			Id: "id1",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Name:       "policy1",
				Severity:   v1.Severity_LOW_SEVERITY,
			},
			Deployment: &v1.ListAlertDeployment{
				ClusterName: "test",
			},
			Time: &timestamp.Timestamp{Seconds: 300},
		},
		{
			Id: "id2",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Container Configuration"},
				Name:       "policy2",
				Severity:   v1.Severity_CRITICAL_SEVERITY,
			},
			Deployment: &v1.ListAlertDeployment{
				ClusterName: "test",
			},
			Time: &timestamp.Timestamp{Seconds: 200},
		},
		{
			Id: "id3",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Name:       "policy1",
				Severity:   v1.Severity_LOW_SEVERITY,
			},
			Deployment: &v1.ListAlertDeployment{
				ClusterName: "prod",
			},
			Time: &timestamp.Timestamp{Seconds: 130},
		},
		{
			Id: "id4",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Privileges Capabilities"},
				Name:       "policy3",
				Severity:   v1.Severity_MEDIUM_SEVERITY,
			},
			Deployment: &v1.ListAlertDeployment{
				ClusterName: "prod",
			},
			Time: &timestamp.Timestamp{Seconds: 120},
		},
		{
			Id: "id5",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance", "Container Configuration"},
				Name:       "policy4",
				Severity:   v1.Severity_HIGH_SEVERITY,
			},
			Deployment: &v1.ListAlertDeployment{
				ClusterName: "prod",
			},
			Time: &timestamp.Timestamp{Seconds: 120},
		},
		{
			Id: "id6",
			Policy: &v1.ListAlertPolicy{
				Categories: []string{"Image Assurance", "Container Configuration"},
				Name:       "policy4",
				Severity:   v1.Severity_HIGH_SEVERITY,
			},
			Deployment: &v1.ListAlertDeployment{
				ClusterName: "test",
			},
			Time: &timestamp.Timestamp{Seconds: 110},
		},
	}

	s.searcher.On("SearchListAlerts", &v1.ParsedSearchRequest{}).Return(fakeListAlertSlice, nil)

	result, err := s.service.GetAlertsCounts(context.Background(), &v1.GetAlertsCountsRequest{Request: &v1.ListAlertsRequest{
		Query: "",
	}, GroupBy: unknownGroupBy})

	s.Equal(status.Error(codes.InvalidArgument, fmt.Sprintf("unknown group by: %v", unknownGroupBy)), err)
	s.Equal((*v1.GetAlertsCountsResponse)(nil), result)
}

func (s *getAlertsCountsTests) TestGetAlertsCountsWhenTheDataAccessLayerFails() {
	s.searcher.On("SearchListAlerts", &v1.ParsedSearchRequest{}).Return(nil, errFake)

	result, err := s.service.GetAlertsCounts(context.Background(), &v1.GetAlertsCountsRequest{Request: &v1.ListAlertsRequest{
		Query: "",
	}})

	s.Equal(status.Error(codes.Internal, "fake error"), err)
	s.Equal((*v1.GetAlertsCountsResponse)(nil), result)
}

type getAlertTimeseriesTests struct {
	suite.Suite

	storage  *storeMocks.Store
	indexer  *indexMocks.Indexer
	searcher *searchMocks.Searcher
	service  Service
}

func (s *getAlertTimeseriesTests) SetupTest() {
	s.storage = new(storeMocks.Store)
	s.indexer = new(indexMocks.Indexer)
	s.searcher = new(searchMocks.Searcher)

	dataStore := datastore.New(s.storage, s.indexer, s.searcher)

	s.service = New(dataStore)
}

func (s *getAlertTimeseriesTests) TestGetAlertTimeseries() {
	alerts := []*v1.ListAlert{
		{
			Id: "id1",
			Time: &timestamp.Timestamp{
				Seconds: 1,
			},
			Stale: true,
			MarkedStale: &timestamp.Timestamp{
				Seconds: 8,
			},
			Deployment: &v1.ListAlertDeployment{ClusterName: "dev"},
			Policy:     &v1.ListAlertPolicy{Severity: v1.Severity_CRITICAL_SEVERITY},
		},
		{
			Id: "id2",
			Time: &timestamp.Timestamp{
				Seconds: 6,
			},
			Deployment: &v1.ListAlertDeployment{ClusterName: "dev"},
			Policy:     &v1.ListAlertPolicy{Severity: v1.Severity_HIGH_SEVERITY},
		},
		{
			Id: "id3",
			Time: &timestamp.Timestamp{
				Seconds: 1,
			},
			Stale: true,
			MarkedStale: &timestamp.Timestamp{
				Seconds: 8,
			},
			Deployment: &v1.ListAlertDeployment{ClusterName: "prod"},
			Policy:     &v1.ListAlertPolicy{Severity: v1.Severity_LOW_SEVERITY},
		},
		{
			Id: "id4",
			Time: &timestamp.Timestamp{
				Seconds: 6,
			},
			Deployment: &v1.ListAlertDeployment{ClusterName: "prod"},
			Policy:     &v1.ListAlertPolicy{Severity: v1.Severity_MEDIUM_SEVERITY},
		},
	}

	expected := &v1.GetAlertTimeseriesResponse{
		Clusters: []*v1.GetAlertTimeseriesResponse_ClusterAlerts{
			{
				Cluster: "dev",
				Severities: []*v1.GetAlertTimeseriesResponse_ClusterAlerts_AlertEvents{
					{
						Severity: v1.Severity_HIGH_SEVERITY,
						Events: []*v1.AlertEvent{
							{
								Time: 6000,
								Id:   "id2",
								Type: v1.Type_CREATED,
							},
						},
					},
					{
						Severity: v1.Severity_CRITICAL_SEVERITY,
						Events: []*v1.AlertEvent{
							{
								Time: 1000,
								Id:   "id1",
								Type: v1.Type_CREATED,
							},
							{
								Time: 8000,
								Id:   "id1",
								Type: v1.Type_REMOVED,
							},
						},
					},
				},
			},
			{
				Cluster: "prod",
				Severities: []*v1.GetAlertTimeseriesResponse_ClusterAlerts_AlertEvents{
					{
						Severity: v1.Severity_LOW_SEVERITY,
						Events: []*v1.AlertEvent{
							{
								Time: 1000,
								Id:   "id3",
								Type: v1.Type_CREATED,
							},
							{
								Time: 8000,
								Id:   "id3",
								Type: v1.Type_REMOVED,
							},
						},
					},
					{
						Severity: v1.Severity_MEDIUM_SEVERITY,
						Events: []*v1.AlertEvent{
							{
								Time: 6000,
								Id:   "id4",
								Type: v1.Type_CREATED,
							},
						},
					},
				},
			},
		},
	}

	s.searcher.On("SearchListAlerts", &v1.ParsedSearchRequest{}).Return(alerts, nil)

	result, err := s.service.GetAlertTimeseries(context.Background(), &v1.ListAlertsRequest{
		Query: "",
	})

	s.NoError(err)
	s.Equal(expected, result)
}

func (s *getAlertTimeseriesTests) TestGetAlertTimeseriesWhenTheDataAccessLayerFails() {
	s.searcher.On("SearchListAlerts", &v1.ParsedSearchRequest{}).Return(nil, errFake)

	result, err := s.service.GetAlertTimeseries(context.Background(), &v1.ListAlertsRequest{
		Query: "",
	})

	s.Equal(status.Error(codes.Internal, "fake error"), err)
	s.Equal((*v1.GetAlertTimeseriesResponse)(nil), result)
}
