package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	dataStoreMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	"github.com/stackrox/rox/central/alert/mappings"
	"github.com/stackrox/rox/central/alerttest"
	baselineMocks "github.com/stackrox/rox/central/processbaseline/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	notifierMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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
	suite.Run(t, new(patchAlertTests))
}

type baseSuite struct {
	suite.Suite

	service Service

	mockCtrl      *gomock.Controller
	datastoreMock *dataStoreMocks.MockDataStore
	notifierMock  *notifierMocks.MockProcessor
	baselineMock  *baselineMocks.MockDataStore
}

func (s *baseSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.notifierMock = notifierMocks.NewMockProcessor(s.mockCtrl)
	s.baselineMock = baselineMocks.NewMockDataStore(s.mockCtrl)
	s.datastoreMock = dataStoreMocks.NewMockDataStore(s.mockCtrl)

	s.service = New(s.datastoreMock, s.baselineMock, s.notifierMock, nil)
}

func (s *baseSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

type getAlertTests struct {
	baseSuite

	fakeResourceByIDRequest *v1.ResourceByID
}

func (s *getAlertTests) SetupTest() {
	s.baseSuite.SetupTest()

	s.fakeResourceByIDRequest = &v1.ResourceByID{
		Id: alerttest.FakeAlertID,
	}
}

func (s *getAlertTests) TestGetAlert() {
	fakeAlert := alerttest.NewFakeAlert()
	fakeContext := context.Background()
	s.datastoreMock.EXPECT().GetAlert(fakeContext, alerttest.FakeAlertID).Return(fakeAlert, true, nil)

	result, err := s.service.GetAlert(fakeContext, s.fakeResourceByIDRequest)

	s.NoError(err)
	s.Equal(fakeAlert, result)
}

func (s *getAlertTests) TestGetAlertWhenTheDataAccessLayerFails() {
	fakeContext := context.Background()

	s.datastoreMock.EXPECT().GetAlert(fakeContext, alerttest.FakeAlertID).Return(alerttest.NewFakeAlert(), false, errFake)

	result, err := s.service.GetAlert(fakeContext, s.fakeResourceByIDRequest)

	s.EqualError(err, "fake error")
	s.Equal((*storage.Alert)(nil), result)
}

func (s *getAlertTests) TestGetAlertWhenAlertIsMissing() {
	fakeContext := context.Background()

	s.datastoreMock.EXPECT().GetAlert(fakeContext, alerttest.FakeAlertID).Return(nil, false, nil)

	result, err := s.service.GetAlert(fakeContext, s.fakeResourceByIDRequest)

	s.EqualError(err, errors.Wrapf(errox.NotFound, "alert with id '%s' does not exist", alerttest.FakeAlertID).Error())
	s.Equal((*storage.Alert)(nil), result)
}

type listAlertsTests struct {
	baseSuite

	fakeListAlertSlice         []*storage.ListAlert
	expectedListAlertsResponse *v1.ListAlertsResponse
}

func (s *listAlertsTests) SetupTest() {
	s.baseSuite.SetupTest()

	s.fakeListAlertSlice = []*storage.ListAlert{
		{
			Id: "id1",
			Time: &types.Timestamp{
				Seconds: 1,
			},
			Policy: &storage.ListAlertPolicy{
				Id: alerttest.FakePolicyID,
			},
		},
		{
			Id: "id2",
			Time: &types.Timestamp{
				Seconds: 2,
			},
			Policy: &storage.ListAlertPolicy{
				Id: alerttest.FakePolicyID,
			},
		},
		{
			Id: "id3",
			Time: &types.Timestamp{
				Seconds: 3,
			},
			Policy: &storage.ListAlertPolicy{
				Id: alerttest.FakePolicyID,
			},
		},
		{
			Id: "id4",
			Time: &types.Timestamp{
				Seconds: 4,
			},
			Policy: &storage.ListAlertPolicy{
				Id: alerttest.FakePolicyID,
			},
		},
	}

	// expectedListAlertsResponse includes the slice of ListAlert objects sorted stably by descending timestamp
	s.expectedListAlertsResponse = &v1.ListAlertsResponse{
		Alerts: s.fakeListAlertSlice,
	}
}

func (s *listAlertsTests) TestListAlerts() {
	fakeQuery := search.NewQueryBuilder().AddStrings(search.DeploymentName, "field1", "field12").AddStrings(search.Category, "field2")
	fakeQueryProto := fakeQuery.ProtoQuery()
	fakeQueryProto.Pagination = &v1.QueryPagination{
		Limit: maxListAlertsReturned,
		SortOptions: []*v1.QuerySortOption{
			paginated.GetViolationTimeSortOption(),
		},
	}
	fakeContext := context.Background()

	s.datastoreMock.EXPECT().SearchListAlerts(fakeContext, fakeQueryProto).Return(s.fakeListAlertSlice, nil)
	result, err := s.service.ListAlerts(fakeContext, &v1.ListAlertsRequest{
		Query: fakeQuery.Query(),
	})

	s.NoError(err)
	s.Equal(s.expectedListAlertsResponse, result)
}

func (s *listAlertsTests) TestListAlertsWhenTheDataLayerFails() {
	fakeContext := context.Background()

	protoQuery := search.NewQueryBuilder().ProtoQuery()
	protoQuery.Pagination = &v1.QueryPagination{
		Limit: maxListAlertsReturned,
		SortOptions: []*v1.QuerySortOption{
			paginated.GetViolationTimeSortOption(),
		},
	}
	s.datastoreMock.EXPECT().SearchListAlerts(fakeContext, protoQuery).Return(nil, errFake)

	result, err := s.service.ListAlerts(fakeContext, &v1.ListAlertsRequest{
		Query: "",
	})

	s.EqualError(err, "fake error")
	s.Equal((*v1.ListAlertsResponse)(nil), result)
}

type getAlertsGroupsTests struct {
	baseSuite
}

func (s *getAlertsGroupsTests) TestGetAlertsGroupForOneCategory() {
	fakeListAlertSlice := []*storage.ListAlert{
		{
			Id: "id1",
			Policy: &storage.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Id:         "id1",
				Name:       "policy1",
				Severity:   storage.Severity_LOW_SEVERITY,
			},
			Time: &types.Timestamp{Seconds: 300},
		},
		{
			Id: "id2",
			Policy: &storage.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Id:         "id2",
				Name:       "policy2",
				Severity:   storage.Severity_HIGH_SEVERITY,
			},
			Time: &types.Timestamp{Seconds: 200},
		},
		{
			Id: "id3",
			Policy: &storage.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Id:         "id1",
				Name:       "policy1",
				Severity:   storage.Severity_LOW_SEVERITY,
			},
			Time: &types.Timestamp{Seconds: 100},
		},
	}

	expected := &v1.GetAlertsGroupResponse{
		AlertsByPolicies: []*v1.GetAlertsGroupResponse_PolicyGroup{
			{
				Policy: &storage.ListAlertPolicy{
					Categories: []string{"Image Assurance"},
					Id:         "id1",
					Name:       "policy1",
					Severity:   storage.Severity_LOW_SEVERITY,
				},
				NumAlerts: 2,
			},
			{
				Policy: &storage.ListAlertPolicy{
					Categories: []string{"Image Assurance"},
					Id:         "id2",
					Name:       "policy2",
					Severity:   storage.Severity_HIGH_SEVERITY,
				},
				NumAlerts: 1,
			},
		},
	}

	s.testGetAlertsGroupFor(fakeListAlertSlice, expected)
}

func (s *getAlertsGroupsTests) TestGetAlertsGroupForMultipleCategories() {
	fakeListAlertSlice := []*storage.ListAlert{
		{
			Id: "id1",
			Policy: &storage.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Id:         "id1",
				Name:       "policy1",
				Severity:   storage.Severity_LOW_SEVERITY,
			},
			Time: &types.Timestamp{Seconds: 300},
		},
		{
			Id: "id2",
			Policy: &storage.ListAlertPolicy{
				Categories: []string{"Image Assurance", "Privileges Capabilities"},
				Id:         "id2",
				Name:       "policy2",
				Severity:   storage.Severity_HIGH_SEVERITY,
			},
			Time: &types.Timestamp{Seconds: 200},
		},
		{
			Id: "id3",
			Policy: &storage.ListAlertPolicy{
				Categories: []string{"Container Configuration"},
				Id:         "id30",
				Name:       "policy30",
				Severity:   storage.Severity_CRITICAL_SEVERITY,
			},
			Time: &types.Timestamp{Seconds: 150},
		},
		{
			Id: "id4",
			Policy: &storage.ListAlertPolicy{
				Categories: []string{"Image Assurance"},
				Id:         "id1",
				Name:       "policy1",
				Severity:   storage.Severity_LOW_SEVERITY,
			},
			Time: &types.Timestamp{Seconds: 100},
		},
	}

	expected := &v1.GetAlertsGroupResponse{
		AlertsByPolicies: []*v1.GetAlertsGroupResponse_PolicyGroup{
			{
				Policy: &storage.ListAlertPolicy{
					Categories: []string{"Image Assurance"},
					Id:         "id1",
					Name:       "policy1",
					Severity:   storage.Severity_LOW_SEVERITY,
				},
				NumAlerts: 2,
			},
			{
				Policy: &storage.ListAlertPolicy{
					Categories: []string{"Image Assurance", "Privileges Capabilities"},
					Id:         "id2",
					Name:       "policy2",
					Severity:   storage.Severity_HIGH_SEVERITY,
				},
				NumAlerts: 1,
			},
			{
				Policy: &storage.ListAlertPolicy{
					Categories: []string{"Container Configuration"},
					Id:         "id30",
					Name:       "policy30",
					Severity:   storage.Severity_CRITICAL_SEVERITY,
				},
				NumAlerts: 1,
			},
		},
	}

	s.testGetAlertsGroupFor(fakeListAlertSlice, expected)
}

func (s *getAlertsGroupsTests) testGetAlertsGroupFor(fakeListAlertSlice []*storage.ListAlert, expected *v1.GetAlertsGroupResponse) {
	fakeContext := context.Background()
	protoQuery := search.NewQueryBuilder().ProtoQuery()
	protoQuery.Pagination = &v1.QueryPagination{
		Limit: math.MaxInt32,
	}
	s.datastoreMock.EXPECT().SearchListAlerts(fakeContext, protoQuery).Return(fakeListAlertSlice, nil)

	result, err := s.service.GetAlertsGroup(fakeContext, &v1.ListAlertsRequest{
		Query: "",
	})

	s.NoError(err)
	s.Equal(expected, result)
}

func (s *getAlertsGroupsTests) TestGetAlertsGroupWhenTheDataAccessLayerFails() {
	fakeContext := context.Background()
	protoQuery := search.NewQueryBuilder().ProtoQuery()
	protoQuery.Pagination = &v1.QueryPagination{
		Limit: math.MaxInt32,
	}
	s.datastoreMock.EXPECT().SearchListAlerts(fakeContext, protoQuery).Return(nil, errFake)

	result, err := s.service.GetAlertsGroup(fakeContext, &v1.ListAlertsRequest{
		Query: "",
	})

	s.EqualError(err, "fake error")
	s.Equal((*v1.GetAlertsGroupResponse)(nil), result)
}

type getAlertsCountsTests struct {
	baseSuite
}

func (s *getAlertsCountsTests) TestGetAlertsCountsWhenAlertsAreNotGrouped() {
	severityField, _ := mappings.OptionsMap.Get(search.Severity.String())
	categoryField, _ := mappings.OptionsMap.Get(search.Category.String())
	clusterField, _ := mappings.OptionsMap.Get(search.Cluster.String())

	fakeSearchResultsSlice := []search.Result{
		{
			ID: "id1",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(1)},
				categoryField.GetFieldPath(): {"Image Assurance"},
				clusterField.GetFieldPath():  {"test"},
			},
		},
		{
			ID: "id2",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(4)},
				categoryField.GetFieldPath(): {"Container Configuration"},
				clusterField.GetFieldPath():  {"test"},
			},
		},
		{
			ID: "id3",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(1)},
				categoryField.GetFieldPath(): {"Image Assurance"},
				clusterField.GetFieldPath():  {"prod"},
			},
		},
		{
			ID: "id4",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(2)},
				categoryField.GetFieldPath(): {"Privileges Capabilities"},
				clusterField.GetFieldPath():  {"prod"},
			},
		},
		{
			ID: "id5",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(3)},
				categoryField.GetFieldPath(): {"Image Assurance", "Container Configuration"},
				clusterField.GetFieldPath():  {"prod"},
			},
		},
		{
			ID: "id6",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(3)},
				categoryField.GetFieldPath(): {"Image Assurance", "Container Configuration"},
				clusterField.GetFieldPath():  {"test"},
			},
		},
	}

	expected := &v1.GetAlertsCountsResponse{
		Groups: []*v1.GetAlertsCountsResponse_AlertGroup{
			{
				Group: "",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					{
						Severity: storage.Severity_LOW_SEVERITY,
						Count:    2,
					},
					{
						Severity: storage.Severity_MEDIUM_SEVERITY,
						Count:    1,
					},
					{
						Severity: storage.Severity_HIGH_SEVERITY,
						Count:    2,
					},
					{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Count:    1,
					},
				},
			},
		},
	}

	s.testGetAlertCounts(fakeSearchResultsSlice, v1.GetAlertsCountsRequest_UNSET, expected)
}

func flagAwareSeverity(i int) string {
	return storage.Severity_name[int32(i)]
}

func (s *getAlertsCountsTests) TestGetAlertsCountsForAlertsGroupedByCategory() {
	severityField, _ := mappings.OptionsMap.Get(search.Severity.String())
	categoryField, _ := mappings.OptionsMap.Get(search.Category.String())
	clusterField, _ := mappings.OptionsMap.Get(search.Cluster.String())

	fakeSearchResultsSlice := []search.Result{
		{
			ID: "id1",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(1)},
				categoryField.GetFieldPath(): {"Image Assurance"},
				clusterField.GetFieldPath():  {"test"},
			},
		},
		{
			ID: "id2",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(4)},
				categoryField.GetFieldPath(): {"Container Configuration"},
				clusterField.GetFieldPath():  {"test"},
			},
		},
		{
			ID: "id3",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(1)},
				categoryField.GetFieldPath(): {"Image Assurance"},
				clusterField.GetFieldPath():  {"prod"},
			},
		},
		{
			ID: "id4",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(2)},
				categoryField.GetFieldPath(): {"Privileges Capabilities"},
				clusterField.GetFieldPath():  {"prod"},
			},
		},
		{
			ID: "id5",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(3)},
				categoryField.GetFieldPath(): {"Image Assurance", "Container Configuration"},
				clusterField.GetFieldPath():  {"prod"},
			},
		},
		{
			ID: "id6",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(3)},
				categoryField.GetFieldPath(): {"Image Assurance", "Container Configuration"},
				clusterField.GetFieldPath():  {"test"},
			},
		},
	}

	expected := &v1.GetAlertsCountsResponse{
		Groups: []*v1.GetAlertsCountsResponse_AlertGroup{
			{
				Group: "Container Configuration",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					{
						Severity: storage.Severity_HIGH_SEVERITY,
						Count:    2,
					},
					{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Count:    1,
					},
				},
			},
			{
				Group: "Image Assurance",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					{
						Severity: storage.Severity_LOW_SEVERITY,
						Count:    2,
					},
					{
						Severity: storage.Severity_HIGH_SEVERITY,
						Count:    2,
					},
				},
			},
			{
				Group: "Privileges Capabilities",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					{
						Severity: storage.Severity_MEDIUM_SEVERITY,
						Count:    1,
					},
				},
			},
		},
	}

	s.testGetAlertCounts(fakeSearchResultsSlice, v1.GetAlertsCountsRequest_CATEGORY, expected)
}

func (s *getAlertsCountsTests) TestGetAlertsCountsForAlertsGroupedByCluster() {
	severityField, _ := mappings.OptionsMap.Get(search.Severity.String())
	categoryField, _ := mappings.OptionsMap.Get(search.Category.String())
	clusterField, _ := mappings.OptionsMap.Get(search.Cluster.String())

	fakeSearchResultsSlice := []search.Result{
		{
			ID: "id1",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(1)},
				categoryField.GetFieldPath(): {"Image Assurance"},
				clusterField.GetFieldPath():  {"test"},
			},
		},
		{
			ID: "id2",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(4)},
				categoryField.GetFieldPath(): {"Container Configuration"},
				clusterField.GetFieldPath():  {"test"},
			},
		},
		{
			ID: "id3",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(1)},
				categoryField.GetFieldPath(): {"Image Assurance"},
				clusterField.GetFieldPath():  {"prod"},
			},
		},
		{
			ID: "id4",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(2)},
				categoryField.GetFieldPath(): {"Privileges Capabilities"},
				clusterField.GetFieldPath():  {"prod"},
			},
		},
		{
			ID: "id5",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(3)},
				categoryField.GetFieldPath(): {"Image Assurance", "Container Configuration"},
				clusterField.GetFieldPath():  {"prod"},
			},
		},
		{
			ID: "id6",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(3)},
				categoryField.GetFieldPath(): {"Image Assurance", "Container Configuration"},
				clusterField.GetFieldPath():  {"test"},
			},
		},
	}

	expected := &v1.GetAlertsCountsResponse{
		Groups: []*v1.GetAlertsCountsResponse_AlertGroup{
			{
				Group: "prod",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					{
						Severity: storage.Severity_LOW_SEVERITY,
						Count:    1,
					},
					{
						Severity: storage.Severity_MEDIUM_SEVERITY,
						Count:    1,
					},
					{
						Severity: storage.Severity_HIGH_SEVERITY,
						Count:    1,
					},
				},
			},
			{
				Group: "test",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					{
						Severity: storage.Severity_LOW_SEVERITY,
						Count:    1,
					},
					{
						Severity: storage.Severity_HIGH_SEVERITY,
						Count:    1,
					},
					{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Count:    1,
					},
				},
			},
		},
	}

	s.testGetAlertCounts(fakeSearchResultsSlice, v1.GetAlertsCountsRequest_CLUSTER, expected)
}

func (s *getAlertsCountsTests) testGetAlertCounts(fakeSearchResultsSlice []search.Result, groupBy v1.GetAlertsCountsRequest_RequestGroup, expected *v1.GetAlertsCountsResponse) {
	fakeContext := context.Background()
	s.datastoreMock.EXPECT().Search(fakeContext, gomock.Any()).Return(fakeSearchResultsSlice, nil)

	result, err := s.service.GetAlertsCounts(fakeContext, &v1.GetAlertsCountsRequest{Request: &v1.ListAlertsRequest{
		Query: "",
	}, GroupBy: groupBy})

	s.NoError(err)
	s.Equal(expected, result)
}

func (s *getAlertsCountsTests) TestGetAlertsCountsWhenTheGroupIsUnknown() {
	const unknownGroupBy = v1.GetAlertsCountsRequest_RequestGroup(-99)
	severityField, _ := mappings.OptionsMap.Get(search.Severity.String())
	categoryField, _ := mappings.OptionsMap.Get(search.Category.String())
	clusterField, _ := mappings.OptionsMap.Get(search.Cluster.String())

	fakeSearchResultsSlice := []search.Result{
		{
			ID: "id1",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(1)},
				categoryField.GetFieldPath(): {"Image Assurance"},
				clusterField.GetFieldPath():  {"test"},
			},
		},
		{
			ID: "id2",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(4)},
				categoryField.GetFieldPath(): {"Container Configuration"},
				clusterField.GetFieldPath():  {"test"},
			},
		},
		{
			ID: "id3",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(1)},
				categoryField.GetFieldPath(): {"Image Assurance"},
				clusterField.GetFieldPath():  {"prod"},
			},
		},
		{
			ID: "id4",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(2)},
				categoryField.GetFieldPath(): {"Privileges Capabilities"},
				clusterField.GetFieldPath():  {"prod"},
			},
		},
		{
			ID: "id5",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(3)},
				categoryField.GetFieldPath(): {"Image Assurance", "Container Configuration"},
				clusterField.GetFieldPath():  {"prod"},
			},
		},
		{
			ID: "id6",
			Matches: map[string][]string{
				severityField.GetFieldPath(): {flagAwareSeverity(3)},
				categoryField.GetFieldPath(): {"Image Assurance", "Container Configuration"},
				clusterField.GetFieldPath():  {"test"},
			},
		},
	}

	fakeContext := context.Background()
	s.datastoreMock.EXPECT().Search(fakeContext, gomock.Any()).Return(fakeSearchResultsSlice, nil)

	result, err := s.service.GetAlertsCounts(fakeContext, &v1.GetAlertsCountsRequest{Request: &v1.ListAlertsRequest{
		Query: "",
	}, GroupBy: unknownGroupBy})

	s.EqualError(err, errors.Wrapf(errox.InvalidArgs, "unknown group by: %v", unknownGroupBy).Error())
	s.Equal((*v1.GetAlertsCountsResponse)(nil), result)
}

func (s *getAlertsCountsTests) TestGetAlertsCountsWhenTheDataAccessLayerFails() {
	fakeContext := context.Background()
	s.datastoreMock.EXPECT().Search(fakeContext, gomock.Any()).Return(nil, errFake)

	result, err := s.service.GetAlertsCounts(fakeContext, &v1.GetAlertsCountsRequest{Request: &v1.ListAlertsRequest{
		Query: "",
	}})

	s.EqualError(err, "fake error")
	s.Equal((*v1.GetAlertsCountsResponse)(nil), result)
}

type getAlertTimeseriesTests struct {
	baseSuite
}

func (s *getAlertTimeseriesTests) TestGetAlertTimeseries() {
	alerts := []*storage.ListAlert{
		{
			Id: "id1",
			Time: &types.Timestamp{
				Seconds: 1,
			},
			State:            storage.ViolationState_RESOLVED,
			Entity:           &storage.ListAlert_Deployment{Deployment: &storage.ListAlertDeployment{ClusterName: "dev"}},
			CommonEntityInfo: &storage.ListAlert_CommonEntityInfo{ClusterName: "dev"},
			Policy:           &storage.ListAlertPolicy{Severity: storage.Severity_CRITICAL_SEVERITY},
		},
		{
			Id: "id2",
			Time: &types.Timestamp{
				Seconds: 6,
			},
			Entity:           &storage.ListAlert_Deployment{Deployment: &storage.ListAlertDeployment{ClusterName: "dev"}},
			CommonEntityInfo: &storage.ListAlert_CommonEntityInfo{ClusterName: "dev"},
			Policy:           &storage.ListAlertPolicy{Severity: storage.Severity_HIGH_SEVERITY},
		},
		{
			Id: "id3",
			Time: &types.Timestamp{
				Seconds: 1,
			},
			State:            storage.ViolationState_RESOLVED,
			Entity:           &storage.ListAlert_Deployment{Deployment: &storage.ListAlertDeployment{ClusterName: "prod"}},
			CommonEntityInfo: &storage.ListAlert_CommonEntityInfo{ClusterName: "prod"},
			Policy:           &storage.ListAlertPolicy{Severity: storage.Severity_LOW_SEVERITY},
		},
		{
			Id: "id4",
			Time: &types.Timestamp{
				Seconds: 6,
			},
			Entity:           &storage.ListAlert_Deployment{Deployment: &storage.ListAlertDeployment{ClusterName: "prod"}},
			CommonEntityInfo: &storage.ListAlert_CommonEntityInfo{ClusterName: "prod"},
			Policy:           &storage.ListAlertPolicy{Severity: storage.Severity_MEDIUM_SEVERITY},
		},
	}

	expected := &v1.GetAlertTimeseriesResponse{
		Clusters: []*v1.GetAlertTimeseriesResponse_ClusterAlerts{
			{
				Cluster: "dev",
				Severities: []*v1.GetAlertTimeseriesResponse_ClusterAlerts_AlertEvents{
					{
						Severity: storage.Severity_HIGH_SEVERITY,
						Events: []*v1.AlertEvent{
							{
								Time: 6000,
								Id:   "id2",
								Type: v1.Type_CREATED,
							},
						},
					},
					{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Events: []*v1.AlertEvent{
							{
								Time: 1000,
								Id:   "id1",
								Type: v1.Type_CREATED,
							},
							{
								Time: 1000,
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
						Severity: storage.Severity_LOW_SEVERITY,
						Events: []*v1.AlertEvent{
							{
								Time: 1000,
								Id:   "id3",
								Type: v1.Type_CREATED,
							},
							{
								Time: 1000,
								Id:   "id3",
								Type: v1.Type_REMOVED,
							},
						},
					},
					{
						Severity: storage.Severity_MEDIUM_SEVERITY,
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
	fakeContext := context.Background()
	protoQuery := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(math.MaxInt32)).ProtoQuery()
	s.datastoreMock.EXPECT().SearchListAlerts(fakeContext, protoQuery).Return(alerts, nil)

	result, err := s.service.GetAlertTimeseries(fakeContext, &v1.ListAlertsRequest{
		Query: "",
	})

	s.NoError(err)
	s.Equal(expected, result)
}

func (s *getAlertTimeseriesTests) TestGetAlertTimeseriesWhenTheDataAccessLayerFails() {
	fakeContext := context.Background()
	protoQuery := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(math.MaxInt32)).ProtoQuery()
	s.datastoreMock.EXPECT().SearchListAlerts(fakeContext, protoQuery).Return(nil, errFake)

	result, err := s.service.GetAlertTimeseries(fakeContext, &v1.ListAlertsRequest{
		Query: "",
	})

	s.EqualError(err, "fake error")
	s.Equal((*v1.GetAlertTimeseriesResponse)(nil), result)
}

type patchAlertTests struct {
	suite.Suite

	storage *dataStoreMocks.MockDataStore
	service Service

	mockCtrl     *gomock.Controller
	notifierMock *notifierMocks.MockProcessor
	baselineMock *baselineMocks.MockDataStore
}

func (s *patchAlertTests) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.storage = dataStoreMocks.NewMockDataStore(s.mockCtrl)
	s.notifierMock = notifierMocks.NewMockProcessor(s.mockCtrl)
	s.baselineMock = baselineMocks.NewMockDataStore(s.mockCtrl)

	s.service = New(s.storage, s.baselineMock, s.notifierMock, nil)
}

func (s *patchAlertTests) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *patchAlertTests) TestSnoozeAlert() {
	fakeAlert := alerttest.NewFakeAlert()
	s.storage.EXPECT().GetAlert(gomock.Any(), alerttest.FakeAlertID).Return(fakeAlert, true, nil)
	snoozeTill, err := types.TimestampProto(time.Now().Add(1 * time.Hour))
	s.NoError(err)
	fakeAlert.SnoozeTill = snoozeTill
	s.storage.EXPECT().UpsertAlert(gomock.Any(), fakeAlert).Return(nil)
	// We should get a notification for the snoozed alert.
	s.notifierMock.EXPECT().ProcessAlert(context.Background(), fakeAlert).Return()
	_, err = s.service.SnoozeAlert(context.Background(), &v1.SnoozeAlertRequest{Id: alerttest.FakeAlertID, SnoozeTill: snoozeTill})
	s.NoError(err)

	s.Equal(fakeAlert.State, storage.ViolationState_SNOOZED)
	s.Equal(fakeAlert.SnoozeTill, snoozeTill)
}

func (s *patchAlertTests) TestSnoozeAlertWithSnoozeTillInThePast() {
	fakeAlert := alerttest.NewFakeAlert()
	s.storage.EXPECT().GetAlert(gomock.Any(), alerttest.FakeAlertID).AnyTimes().Return(fakeAlert, true, nil)

	snoozeTill, err := types.TimestampProto(time.Now().Add(-1 * time.Hour))
	s.NoError(err)
	_, err = s.service.SnoozeAlert(context.Background(), &v1.SnoozeAlertRequest{Id: alerttest.FakeAlertID, SnoozeTill: snoozeTill})
	s.EqualError(err, errors.Wrap(errox.InvalidArgs, badSnoozeErrorMsg).Error())
}

func (s *patchAlertTests) TestResolveAlert() {
	fakeAlert := alerttest.NewFakeAlert()
	s.storage.EXPECT().GetAlert(gomock.Any(), alerttest.FakeAlertID).Return(fakeAlert, true, nil)
	fakeAlert.State = storage.ViolationState_RESOLVED
	s.storage.EXPECT().UpsertAlert(gomock.Any(), fakeAlert).Return(nil)
	// We should get a notification for the resolved alert.
	s.notifierMock.EXPECT().ProcessAlert(context.Background(), fakeAlert).Return()

	_, err := s.service.ResolveAlert(context.Background(), &v1.ResolveAlertRequest{Id: alerttest.FakeAlertID})
	s.NoError(err)
	s.Equal(fakeAlert.State, storage.ViolationState_RESOLVED)
}

func (s *baseSuite) TestDeleteAlerts() {
	errorCases := []struct {
		request *v1.DeleteAlertsRequest
	}{
		{
			request: &v1.DeleteAlertsRequest{},
		},
		{
			request: &v1.DeleteAlertsRequest{
				Query: &v1.RawQuery{},
			},
		},
		{
			request: &v1.DeleteAlertsRequest{
				Query: &v1.RawQuery{
					Query: search.NewQueryBuilder().AddStrings(search.DeploymentName, "lol").Query(),
				},
			},
		},
		{
			request: &v1.DeleteAlertsRequest{
				Query: &v1.RawQuery{
					Query: search.NewQueryBuilder().AddStrings(search.DeploymentName, "lol").Query(),
				},
			},
		},
		{
			request: &v1.DeleteAlertsRequest{
				Query: &v1.RawQuery{
					Query: search.NewQueryBuilder().AddStrings(search.ViolationState, "ACTIVE").Query(),
				},
			},
		},
	}

	for _, e := range errorCases {
		s.T().Run(s.T().Name(), func(t *testing.T) {
			_, err := s.service.DeleteAlerts(context.Background(), e.request)
			assert.Error(t, err)
		})
	}

	expectedQueryBuilder := search.NewQueryBuilder().
		AddStrings(search.DeploymentName, "deployment").
		AddStrings(search.ViolationState, storage.ViolationState_RESOLVED.String())
	expectedQuery := expectedQueryBuilder.ProtoQuery()
	expectedQuery.Pagination = &v1.QueryPagination{
		Limit: math.MaxInt32,
	}

	s.datastoreMock.EXPECT().Search(context.Background(), expectedQuery).Return([]search.Result{}, nil)

	_, err := s.service.DeleteAlerts(context.Background(), &v1.DeleteAlertsRequest{
		Query: &v1.RawQuery{
			Query: expectedQueryBuilder.Query(),
		},
	})
	s.NoError(err)
}
