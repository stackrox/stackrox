package service

import (
	"context"
	"math"
	"testing"

	"github.com/pkg/errors"
	dataStoreMocks "github.com/stackrox/rox/central/alert/datastore/mocks"
	"github.com/stackrox/rox/central/alert/mappings"
	"github.com/stackrox/rox/central/alerttest"
	baselineMocks "github.com/stackrox/rox/central/processbaseline/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	notifierMocks "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
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

	rbid := &v1.ResourceByID{}
	rbid.SetId(alerttest.FakeAlertID)
	s.fakeResourceByIDRequest = rbid
}

func (s *getAlertTests) TestGetAlert() {
	fakeAlert := alerttest.NewFakeAlert()
	fakeContext := context.Background()
	s.datastoreMock.EXPECT().GetAlert(fakeContext, alerttest.FakeAlertID).Return(fakeAlert, true, nil)

	result, err := s.service.GetAlert(fakeContext, s.fakeResourceByIDRequest)

	s.NoError(err)
	protoassert.Equal(s.T(), fakeAlert, result)
}

func (s *getAlertTests) TestGetAlertWhenTheDataAccessLayerFails() {
	fakeContext := context.Background()

	s.datastoreMock.EXPECT().GetAlert(fakeContext, alerttest.FakeAlertID).Return(alerttest.NewFakeAlert(), false, errFake)

	result, err := s.service.GetAlert(fakeContext, s.fakeResourceByIDRequest)

	s.EqualError(err, "fake error")
	s.Nil(result)
}

func (s *getAlertTests) TestGetAlertWhenAlertIsMissing() {
	fakeContext := context.Background()

	s.datastoreMock.EXPECT().GetAlert(fakeContext, alerttest.FakeAlertID).Return(nil, false, nil)

	result, err := s.service.GetAlert(fakeContext, s.fakeResourceByIDRequest)

	s.EqualError(err, errors.Wrapf(errox.NotFound, "alert with id '%s' does not exist", alerttest.FakeAlertID).Error())
	s.Nil(result)
}

type listAlertsTests struct {
	baseSuite

	fakeListAlertSlice         []*storage.ListAlert
	expectedListAlertsResponse *v1.ListAlertsResponse
}

func (s *listAlertsTests) SetupTest() {
	s.baseSuite.SetupTest()

	s.fakeListAlertSlice = []*storage.ListAlert{
		storage.ListAlert_builder{
			Id:   "id1",
			Time: protocompat.GetProtoTimestampFromSeconds(1),

			Policy: storage.ListAlertPolicy_builder{
				Id: alerttest.FakePolicyID,
			}.Build(),
		}.Build(),
		storage.ListAlert_builder{
			Id:   "id2",
			Time: protocompat.GetProtoTimestampFromSeconds(2),

			Policy: storage.ListAlertPolicy_builder{
				Id: alerttest.FakePolicyID,
			}.Build(),
		}.Build(),
		storage.ListAlert_builder{
			Id:   "id3",
			Time: protocompat.GetProtoTimestampFromSeconds(3),

			Policy: storage.ListAlertPolicy_builder{
				Id: alerttest.FakePolicyID,
			}.Build(),
		}.Build(),
		storage.ListAlert_builder{
			Id:   "id4",
			Time: protocompat.GetProtoTimestampFromSeconds(4),

			Policy: storage.ListAlertPolicy_builder{
				Id: alerttest.FakePolicyID,
			}.Build(),
		}.Build(),
	}

	// expectedListAlertsResponse includes the slice of ListAlert objects sorted stably by descending timestamp
	lar := &v1.ListAlertsResponse{}
	lar.SetAlerts(s.fakeListAlertSlice)
	s.expectedListAlertsResponse = lar
}

func (s *listAlertsTests) TestListAlerts() {
	fakeQuery := search.NewQueryBuilder().AddStrings(search.DeploymentName, "field1", "field12").AddStrings(search.Category, "field2")
	fakeQueryProto := fakeQuery.ProtoQuery()
	qp := &v1.QueryPagination{}
	qp.SetLimit(maxListAlertsReturned)
	qp.SetSortOptions([]*v1.QuerySortOption{
		paginated.GetViolationTimeSortOption(),
	})
	fakeQueryProto.SetPagination(qp)
	fakeContext := context.Background()

	s.datastoreMock.EXPECT().SearchListAlerts(fakeContext, fakeQueryProto, true).Return(s.fakeListAlertSlice, nil)
	lar := &v1.ListAlertsRequest{}
	lar.SetQuery(fakeQuery.Query())
	result, err := s.service.ListAlerts(fakeContext, lar)

	s.NoError(err)
	protoassert.Equal(s.T(), s.expectedListAlertsResponse, result)
}

func (s *listAlertsTests) TestListAlertsWhenTheDataLayerFails() {
	fakeContext := context.Background()

	protoQuery := search.NewQueryBuilder().ProtoQuery()
	qp := &v1.QueryPagination{}
	qp.SetLimit(maxListAlertsReturned)
	qp.SetSortOptions([]*v1.QuerySortOption{
		paginated.GetViolationTimeSortOption(),
	})
	protoQuery.SetPagination(qp)
	s.datastoreMock.EXPECT().SearchListAlerts(fakeContext, protoQuery, true).Return(nil, errFake)

	lar := &v1.ListAlertsRequest{}
	lar.SetQuery("")
	result, err := s.service.ListAlerts(fakeContext, lar)

	s.EqualError(err, "fake error")
	s.Nil(result)
}

type getAlertsGroupsTests struct {
	baseSuite
}

func (s *getAlertsGroupsTests) TestGetAlertsGroupForOneCategory() {
	fakeListAlertSlice := []*storage.ListAlert{
		storage.ListAlert_builder{
			Id: "id1",
			Policy: storage.ListAlertPolicy_builder{
				Categories: []string{"Image Assurance"},
				Id:         "id1",
				Name:       "policy1",
				Severity:   storage.Severity_LOW_SEVERITY,
			}.Build(),
			Time: protocompat.GetProtoTimestampFromSeconds(300),
		}.Build(),
		storage.ListAlert_builder{
			Id: "id2",
			Policy: storage.ListAlertPolicy_builder{
				Categories: []string{"Image Assurance"},
				Id:         "id2",
				Name:       "policy2",
				Severity:   storage.Severity_HIGH_SEVERITY,
			}.Build(),
			Time: protocompat.GetProtoTimestampFromSeconds(200),
		}.Build(),
		storage.ListAlert_builder{
			Id: "id3",
			Policy: storage.ListAlertPolicy_builder{
				Categories: []string{"Image Assurance"},
				Id:         "id1",
				Name:       "policy1",
				Severity:   storage.Severity_LOW_SEVERITY,
			}.Build(),
			Time: protocompat.GetProtoTimestampFromSeconds(100),
		}.Build(),
	}

	expected := v1.GetAlertsGroupResponse_builder{
		AlertsByPolicies: []*v1.GetAlertsGroupResponse_PolicyGroup{
			v1.GetAlertsGroupResponse_PolicyGroup_builder{
				Policy: storage.ListAlertPolicy_builder{
					Categories: []string{"Image Assurance"},
					Id:         "id1",
					Name:       "policy1",
					Severity:   storage.Severity_LOW_SEVERITY,
				}.Build(),
				NumAlerts: 2,
			}.Build(),
			v1.GetAlertsGroupResponse_PolicyGroup_builder{
				Policy: storage.ListAlertPolicy_builder{
					Categories: []string{"Image Assurance"},
					Id:         "id2",
					Name:       "policy2",
					Severity:   storage.Severity_HIGH_SEVERITY,
				}.Build(),
				NumAlerts: 1,
			}.Build(),
		},
	}.Build()

	s.testGetAlertsGroupFor(fakeListAlertSlice, expected)
}

func (s *getAlertsGroupsTests) TestGetAlertsGroupForMultipleCategories() {
	fakeListAlertSlice := []*storage.ListAlert{
		storage.ListAlert_builder{
			Id: "id1",
			Policy: storage.ListAlertPolicy_builder{
				Categories: []string{"Image Assurance"},
				Id:         "id1",
				Name:       "policy1",
				Severity:   storage.Severity_LOW_SEVERITY,
			}.Build(),
			Time: protocompat.GetProtoTimestampFromSeconds(300),
		}.Build(),
		storage.ListAlert_builder{
			Id: "id2",
			Policy: storage.ListAlertPolicy_builder{
				Categories: []string{"Image Assurance", "Privileges Capabilities"},
				Id:         "id2",
				Name:       "policy2",
				Severity:   storage.Severity_HIGH_SEVERITY,
			}.Build(),
			Time: protocompat.GetProtoTimestampFromSeconds(200),
		}.Build(),
		storage.ListAlert_builder{
			Id: "id3",
			Policy: storage.ListAlertPolicy_builder{
				Categories: []string{"Container Configuration"},
				Id:         "id30",
				Name:       "policy30",
				Severity:   storage.Severity_CRITICAL_SEVERITY,
			}.Build(),
			Time: protocompat.GetProtoTimestampFromSeconds(150),
		}.Build(),
		storage.ListAlert_builder{
			Id: "id4",
			Policy: storage.ListAlertPolicy_builder{
				Categories: []string{"Image Assurance"},
				Id:         "id1",
				Name:       "policy1",
				Severity:   storage.Severity_LOW_SEVERITY,
			}.Build(),
			Time: protocompat.GetProtoTimestampFromSeconds(100),
		}.Build(),
	}

	expected := v1.GetAlertsGroupResponse_builder{
		AlertsByPolicies: []*v1.GetAlertsGroupResponse_PolicyGroup{
			v1.GetAlertsGroupResponse_PolicyGroup_builder{
				Policy: storage.ListAlertPolicy_builder{
					Categories: []string{"Image Assurance"},
					Id:         "id1",
					Name:       "policy1",
					Severity:   storage.Severity_LOW_SEVERITY,
				}.Build(),
				NumAlerts: 2,
			}.Build(),
			v1.GetAlertsGroupResponse_PolicyGroup_builder{
				Policy: storage.ListAlertPolicy_builder{
					Categories: []string{"Image Assurance", "Privileges Capabilities"},
					Id:         "id2",
					Name:       "policy2",
					Severity:   storage.Severity_HIGH_SEVERITY,
				}.Build(),
				NumAlerts: 1,
			}.Build(),
			v1.GetAlertsGroupResponse_PolicyGroup_builder{
				Policy: storage.ListAlertPolicy_builder{
					Categories: []string{"Container Configuration"},
					Id:         "id30",
					Name:       "policy30",
					Severity:   storage.Severity_CRITICAL_SEVERITY,
				}.Build(),
				NumAlerts: 1,
			}.Build(),
		},
	}.Build()

	s.testGetAlertsGroupFor(fakeListAlertSlice, expected)
}

func (s *getAlertsGroupsTests) testGetAlertsGroupFor(fakeListAlertSlice []*storage.ListAlert, expected *v1.GetAlertsGroupResponse) {
	fakeContext := context.Background()
	protoQuery := search.NewQueryBuilder().ProtoQuery()
	qp := &v1.QueryPagination{}
	qp.SetLimit(math.MaxInt32)
	protoQuery.SetPagination(qp)
	s.datastoreMock.EXPECT().SearchListAlerts(fakeContext, protoQuery, true).Return(fakeListAlertSlice, nil)

	lar := &v1.ListAlertsRequest{}
	lar.SetQuery("")
	result, err := s.service.GetAlertsGroup(fakeContext, lar)

	s.NoError(err)
	protoassert.Equal(s.T(), expected, result)
}

func (s *getAlertsGroupsTests) TestGetAlertsGroupWhenTheDataAccessLayerFails() {
	fakeContext := context.Background()
	protoQuery := search.NewQueryBuilder().ProtoQuery()
	qp := &v1.QueryPagination{}
	qp.SetLimit(math.MaxInt32)
	protoQuery.SetPagination(qp)
	s.datastoreMock.EXPECT().SearchListAlerts(fakeContext, protoQuery, true).Return(nil, errFake)

	lar := &v1.ListAlertsRequest{}
	lar.SetQuery("")
	result, err := s.service.GetAlertsGroup(fakeContext, lar)

	s.EqualError(err, "fake error")
	s.Nil(result)
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

	expected := v1.GetAlertsCountsResponse_builder{
		Groups: []*v1.GetAlertsCountsResponse_AlertGroup{
			v1.GetAlertsCountsResponse_AlertGroup_builder{
				Group: "",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_LOW_SEVERITY,
						Count:    2,
					}.Build(),
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_MEDIUM_SEVERITY,
						Count:    1,
					}.Build(),
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_HIGH_SEVERITY,
						Count:    2,
					}.Build(),
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Count:    1,
					}.Build(),
				},
			}.Build(),
		},
	}.Build()

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

	expected := v1.GetAlertsCountsResponse_builder{
		Groups: []*v1.GetAlertsCountsResponse_AlertGroup{
			v1.GetAlertsCountsResponse_AlertGroup_builder{
				Group: "Container Configuration",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_HIGH_SEVERITY,
						Count:    2,
					}.Build(),
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Count:    1,
					}.Build(),
				},
			}.Build(),
			v1.GetAlertsCountsResponse_AlertGroup_builder{
				Group: "Image Assurance",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_LOW_SEVERITY,
						Count:    2,
					}.Build(),
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_HIGH_SEVERITY,
						Count:    2,
					}.Build(),
				},
			}.Build(),
			v1.GetAlertsCountsResponse_AlertGroup_builder{
				Group: "Privileges Capabilities",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_MEDIUM_SEVERITY,
						Count:    1,
					}.Build(),
				},
			}.Build(),
		},
	}.Build()

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

	expected := v1.GetAlertsCountsResponse_builder{
		Groups: []*v1.GetAlertsCountsResponse_AlertGroup{
			v1.GetAlertsCountsResponse_AlertGroup_builder{
				Group: "prod",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_LOW_SEVERITY,
						Count:    1,
					}.Build(),
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_MEDIUM_SEVERITY,
						Count:    1,
					}.Build(),
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_HIGH_SEVERITY,
						Count:    1,
					}.Build(),
				},
			}.Build(),
			v1.GetAlertsCountsResponse_AlertGroup_builder{
				Group: "test",
				Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_LOW_SEVERITY,
						Count:    1,
					}.Build(),
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_HIGH_SEVERITY,
						Count:    1,
					}.Build(),
					v1.GetAlertsCountsResponse_AlertGroup_AlertCounts_builder{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Count:    1,
					}.Build(),
				},
			}.Build(),
		},
	}.Build()

	s.testGetAlertCounts(fakeSearchResultsSlice, v1.GetAlertsCountsRequest_CLUSTER, expected)
}

func (s *getAlertsCountsTests) testGetAlertCounts(fakeSearchResultsSlice []search.Result, groupBy v1.GetAlertsCountsRequest_RequestGroup, expected *v1.GetAlertsCountsResponse) {
	fakeContext := context.Background()
	s.datastoreMock.EXPECT().Search(fakeContext, gomock.Any(), true).Return(fakeSearchResultsSlice, nil)

	lar := &v1.ListAlertsRequest{}
	lar.SetQuery("")
	gacr := &v1.GetAlertsCountsRequest{}
	gacr.SetRequest(lar)
	gacr.SetGroupBy(groupBy)
	result, err := s.service.GetAlertsCounts(fakeContext, gacr)

	s.NoError(err)
	protoassert.Equal(s.T(), expected, result)
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
	s.datastoreMock.EXPECT().Search(fakeContext, gomock.Any(), true).Return(fakeSearchResultsSlice, nil)

	lar := &v1.ListAlertsRequest{}
	lar.SetQuery("")
	gacr := &v1.GetAlertsCountsRequest{}
	gacr.SetRequest(lar)
	gacr.SetGroupBy(unknownGroupBy)
	result, err := s.service.GetAlertsCounts(fakeContext, gacr)

	s.EqualError(err, errors.Wrapf(errox.InvalidArgs, "unknown group by: %v", unknownGroupBy).Error())
	s.Nil(result)
}

func (s *getAlertsCountsTests) TestGetAlertsCountsWhenTheDataAccessLayerFails() {
	fakeContext := context.Background()
	s.datastoreMock.EXPECT().Search(fakeContext, gomock.Any(), true).Return(nil, errFake)

	lar := &v1.ListAlertsRequest{}
	lar.SetQuery("")
	gacr := &v1.GetAlertsCountsRequest{}
	gacr.SetRequest(lar)
	result, err := s.service.GetAlertsCounts(fakeContext, gacr)

	s.EqualError(err, "fake error")
	s.Nil(result)
}

type getAlertTimeseriesTests struct {
	baseSuite
}

func (s *getAlertTimeseriesTests) TestGetAlertTimeseries() {
	alerts := []*storage.ListAlert{
		storage.ListAlert_builder{
			Id:   "id1",
			Time: protocompat.GetProtoTimestampFromSeconds(1),

			State:            storage.ViolationState_RESOLVED,
			Deployment:       storage.ListAlertDeployment_builder{ClusterName: "dev"}.Build(),
			CommonEntityInfo: storage.ListAlert_CommonEntityInfo_builder{ClusterName: "dev"}.Build(),
			Policy:           storage.ListAlertPolicy_builder{Severity: storage.Severity_CRITICAL_SEVERITY}.Build(),
		}.Build(),
		storage.ListAlert_builder{
			Id:   "id2",
			Time: protocompat.GetProtoTimestampFromSeconds(6),

			Deployment:       storage.ListAlertDeployment_builder{ClusterName: "dev"}.Build(),
			CommonEntityInfo: storage.ListAlert_CommonEntityInfo_builder{ClusterName: "dev"}.Build(),
			Policy:           storage.ListAlertPolicy_builder{Severity: storage.Severity_HIGH_SEVERITY}.Build(),
		}.Build(),
		storage.ListAlert_builder{
			Id:   "id3",
			Time: protocompat.GetProtoTimestampFromSeconds(1),

			State:            storage.ViolationState_RESOLVED,
			Deployment:       storage.ListAlertDeployment_builder{ClusterName: "prod"}.Build(),
			CommonEntityInfo: storage.ListAlert_CommonEntityInfo_builder{ClusterName: "prod"}.Build(),
			Policy:           storage.ListAlertPolicy_builder{Severity: storage.Severity_LOW_SEVERITY}.Build(),
		}.Build(),
		storage.ListAlert_builder{
			Id:   "id4",
			Time: protocompat.GetProtoTimestampFromSeconds(6),

			Deployment:       storage.ListAlertDeployment_builder{ClusterName: "prod"}.Build(),
			CommonEntityInfo: storage.ListAlert_CommonEntityInfo_builder{ClusterName: "prod"}.Build(),
			Policy:           storage.ListAlertPolicy_builder{Severity: storage.Severity_MEDIUM_SEVERITY}.Build(),
		}.Build(),
	}

	expected := v1.GetAlertTimeseriesResponse_builder{
		Clusters: []*v1.GetAlertTimeseriesResponse_ClusterAlerts{
			v1.GetAlertTimeseriesResponse_ClusterAlerts_builder{
				Cluster: "dev",
				Severities: []*v1.GetAlertTimeseriesResponse_ClusterAlerts_AlertEvents{
					v1.GetAlertTimeseriesResponse_ClusterAlerts_AlertEvents_builder{
						Severity: storage.Severity_HIGH_SEVERITY,
						Events: []*v1.AlertEvent{
							v1.AlertEvent_builder{
								Time: 6000,
								Id:   "id2",
								Type: v1.Type_CREATED,
							}.Build(),
						},
					}.Build(),
					v1.GetAlertTimeseriesResponse_ClusterAlerts_AlertEvents_builder{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Events: []*v1.AlertEvent{
							v1.AlertEvent_builder{
								Time: 1000,
								Id:   "id1",
								Type: v1.Type_CREATED,
							}.Build(),
							v1.AlertEvent_builder{
								Time: 1000,
								Id:   "id1",
								Type: v1.Type_REMOVED,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
			v1.GetAlertTimeseriesResponse_ClusterAlerts_builder{
				Cluster: "prod",
				Severities: []*v1.GetAlertTimeseriesResponse_ClusterAlerts_AlertEvents{
					v1.GetAlertTimeseriesResponse_ClusterAlerts_AlertEvents_builder{
						Severity: storage.Severity_LOW_SEVERITY,
						Events: []*v1.AlertEvent{
							v1.AlertEvent_builder{
								Time: 1000,
								Id:   "id3",
								Type: v1.Type_CREATED,
							}.Build(),
							v1.AlertEvent_builder{
								Time: 1000,
								Id:   "id3",
								Type: v1.Type_REMOVED,
							}.Build(),
						},
					}.Build(),
					v1.GetAlertTimeseriesResponse_ClusterAlerts_AlertEvents_builder{
						Severity: storage.Severity_MEDIUM_SEVERITY,
						Events: []*v1.AlertEvent{
							v1.AlertEvent_builder{
								Time: 6000,
								Id:   "id4",
								Type: v1.Type_CREATED,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		},
	}.Build()
	fakeContext := context.Background()
	protoQuery := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(math.MaxInt32)).ProtoQuery()
	s.datastoreMock.EXPECT().SearchListAlerts(fakeContext, protoQuery, true).Return(alerts, nil)

	lar := &v1.ListAlertsRequest{}
	lar.SetQuery("")
	result, err := s.service.GetAlertTimeseries(fakeContext, lar)

	s.NoError(err)
	protoassert.Equal(s.T(), expected, result)
}

func (s *getAlertTimeseriesTests) TestGetAlertTimeseriesWhenTheDataAccessLayerFails() {
	fakeContext := context.Background()
	protoQuery := search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(math.MaxInt32)).ProtoQuery()
	s.datastoreMock.EXPECT().SearchListAlerts(fakeContext, protoQuery, true).Return(nil, errFake)

	lar := &v1.ListAlertsRequest{}
	lar.SetQuery("")
	result, err := s.service.GetAlertTimeseries(fakeContext, lar)

	s.EqualError(err, "fake error")
	s.Nil(result)
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

func (s *patchAlertTests) TestResolveAlert() {
	fakeAlert := alerttest.NewFakeAlert()
	s.storage.EXPECT().GetAlert(gomock.Any(), alerttest.FakeAlertID).Return(fakeAlert, true, nil)
	fakeAlert.SetState(storage.ViolationState_RESOLVED)
	s.storage.EXPECT().UpsertAlert(gomock.Any(), fakeAlert).Return(nil)
	// We should get a notification for the resolved alert.
	s.notifierMock.EXPECT().ProcessAlert(context.Background(), fakeAlert).Return()

	rar := &v1.ResolveAlertRequest{}
	rar.SetId(alerttest.FakeAlertID)
	_, err := s.service.ResolveAlert(context.Background(), rar)
	s.NoError(err)
	s.Equal(fakeAlert.GetState(), storage.ViolationState_RESOLVED)
}

func (s *baseSuite) TestDeleteAlerts() {
	errorCases := []struct {
		request *v1.DeleteAlertsRequest
	}{
		{
			request: &v1.DeleteAlertsRequest{},
		},
		{
			request: v1.DeleteAlertsRequest_builder{
				Query: &v1.RawQuery{},
			}.Build(),
		},
		{
			request: v1.DeleteAlertsRequest_builder{
				Query: v1.RawQuery_builder{
					Query: search.NewQueryBuilder().AddStrings(search.DeploymentName, "lol").Query(),
				}.Build(),
			}.Build(),
		},
		{
			request: v1.DeleteAlertsRequest_builder{
				Query: v1.RawQuery_builder{
					Query: search.NewQueryBuilder().AddStrings(search.DeploymentName, "lol").Query(),
				}.Build(),
			}.Build(),
		},
		{
			request: v1.DeleteAlertsRequest_builder{
				Query: v1.RawQuery_builder{
					Query: search.NewQueryBuilder().AddStrings(search.ViolationState, "ACTIVE").Query(),
				}.Build(),
			}.Build(),
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
	qp := &v1.QueryPagination{}
	qp.SetLimit(math.MaxInt32)
	expectedQuery.SetPagination(qp)

	s.datastoreMock.EXPECT().Search(context.Background(), expectedQuery, true).Return([]search.Result{}, nil)

	rawQuery := &v1.RawQuery{}
	rawQuery.SetQuery(expectedQueryBuilder.Query())
	dar := &v1.DeleteAlertsRequest{}
	dar.SetQuery(rawQuery)
	_, err := s.service.DeleteAlerts(context.Background(), dar)
	s.NoError(err)
}
