package service

import (
	"context"
	_ "embed"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/pkg/errors"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	lifecycleMocks "github.com/stackrox/rox/central/detection/lifecycle/mocks"
	"github.com/stackrox/rox/central/policy/datastore/mocks"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	mitreMocks "github.com/stackrox/rox/pkg/mitre/datastore/mocks"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	mockRequestOneID = &v1.ExportPoliciesRequest{
		PolicyIds: []string{"Joseph Rules"},
	}
	mockRequestTwoIDs = &v1.ExportPoliciesRequest{
		PolicyIds: []string{"Joseph Rules", "abcd"},
	}
)

func TestPolicyService(t *testing.T) {
	suite.Run(t, new(PolicyServiceTestSuite))
}

type PolicyServiceTestSuite struct {
	suite.Suite
	policies              *mocks.MockDataStore
	clusters              *clusterMocks.MockDataStore
	mitreVectorStore      *mitreMocks.MockAttackReadOnlyDataStore
	mockLifecycleManager  *lifecycleMocks.MockManager
	mockConnectionManager *connectionMocks.MockManager
	tested                Service

	mockCtrl *gomock.Controller
}

func (s *PolicyServiceTestSuite) SetupTest() {

	s.mockCtrl = gomock.NewController(s.T())

	s.policies = mocks.NewMockDataStore(s.mockCtrl)

	s.clusters = clusterMocks.NewMockDataStore(s.mockCtrl)

	s.mockLifecycleManager = lifecycleMocks.NewMockManager(s.mockCtrl)
	s.mockConnectionManager = connectionMocks.NewMockManager(s.mockCtrl)
	s.mitreVectorStore = mitreMocks.NewMockAttackReadOnlyDataStore(s.mockCtrl)

	s.tested = New(
		s.policies,
		s.clusters,
		nil,
		nil,
		nil,
		s.mitreVectorStore,
		nil,
		s.mockLifecycleManager,
		nil,
		nil,
		s.mockConnectionManager,
	)
}

func (s *PolicyServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *PolicyServiceTestSuite) compareErrorsToExpected(expectedErrors []*v1.PolicyOperationError, apiError error) {
	apiStatus, ok := status.FromError(apiError)
	s.Require().True(ok)
	details := apiStatus.Details()
	s.Len(details, 1)
	exportErrors, ok := details[0].(*v1.PolicyOperationErrorList)
	s.Require().True(ok)
	// actual errors == expected errors ignoring order
	s.Len(exportErrors.GetErrors(), len(expectedErrors))
	for _, expected := range expectedErrors {
		protoassert.SliceContains(s.T(), exportErrors.GetErrors(), expected)
	}
}

func makeError(errorID, errorString string) *v1.PolicyOperationError {
	pe := &v1.PolicyError{}
	pe.SetError(errorString)
	poe := &v1.PolicyOperationError{}
	poe.SetPolicyId(errorID)
	poe.SetError(pe)
	return poe
}

func (s *PolicyServiceTestSuite) TestExportInvalidIDFails() {
	ctx := context.Background()
	mockErrors := []*v1.PolicyOperationError{
		makeError(mockRequestOneID.GetPolicyIds()[0], "not found"),
	}
	s.policies.EXPECT().GetPolicies(ctx, mockRequestOneID.GetPolicyIds()).Return(make([]*storage.Policy, 0), []int{0}, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestOneID)
	s.Nil(resp)
	s.Error(err)
	s.compareErrorsToExpected(mockErrors, err)
}

func (s *PolicyServiceTestSuite) TestExportValidIDSucceeds() {
	ctx := context.Background()
	mockPolicy := &storage.Policy{}
	mockPolicy.SetId(mockRequestOneID.GetPolicyIds()[0])
	s.policies.EXPECT().GetPolicies(ctx, mockRequestOneID.GetPolicyIds()).Return([]*storage.Policy{mockPolicy}, nil, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestOneID)
	s.NoError(err)
	s.NotNil(resp)
	s.Len(resp.GetPolicies(), 1)
	protoassert.Equal(s.T(), mockPolicy, resp.GetPolicies()[0])
}

func (s *PolicyServiceTestSuite) TestExportMixedSuccessAndMissing() {
	ctx := context.Background()
	mockPolicy := &storage.Policy{}
	mockPolicy.SetId(mockRequestTwoIDs.GetPolicyIds()[0])
	mockErrors := []*v1.PolicyOperationError{
		makeError(mockRequestTwoIDs.GetPolicyIds()[1], "not found"),
	}
	s.policies.EXPECT().GetPolicies(ctx, mockRequestTwoIDs.GetPolicyIds()).Return([]*storage.Policy{mockPolicy}, []int{1}, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestTwoIDs)
	s.Nil(resp)
	s.Error(err)
	s.compareErrorsToExpected(mockErrors, err)
}

func (s *PolicyServiceTestSuite) TestExportMultipleFailures() {
	ctx := context.Background()
	mockErrors := []*v1.PolicyOperationError{
		makeError(mockRequestTwoIDs.GetPolicyIds()[0], "not found"),
		makeError(mockRequestTwoIDs.GetPolicyIds()[1], "not found"),
	}
	s.policies.EXPECT().GetPolicies(ctx, mockRequestTwoIDs.GetPolicyIds()).Return(make([]*storage.Policy, 0), []int{0, 1}, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestTwoIDs)
	s.Nil(resp)
	s.Error(err)
	s.compareErrorsToExpected(mockErrors, err)
}

func (s *PolicyServiceTestSuite) TestExportedPolicyHasNoSortFields() {
	ctx := context.Background()
	mockPolicy := &storage.Policy{}
	mockPolicy.SetId(mockRequestOneID.GetPolicyIds()[0])
	mockPolicy.SetSORTName("abc")
	mockPolicy.SetSORTLifecycleStage("def")
	expectedPolicy := &storage.Policy{}
	expectedPolicy.SetId(mockRequestOneID.GetPolicyIds()[0])
	s.policies.EXPECT().GetPolicies(ctx, mockRequestOneID.GetPolicyIds()).Return([]*storage.Policy{mockPolicy}, nil, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestOneID)
	s.NoError(err)
	s.NotNil(resp)
	s.Len(resp.GetPolicies(), 1)
	protoassert.Equal(s.T(), expectedPolicy, resp.GetPolicies()[0])
}

func (s *PolicyServiceTestSuite) TestPoliciesHaveNoUnexpectedSORTFields() {
	expectedSORTFields := set.NewStringSet("SORTLifecycleStage", "SORTEnforcement", "SORTName")
	var policy *storage.Policy
	policyType := reflect.TypeOf(policy).Elem()
	numFields := policyType.NumField()
	for i := 0; i < numFields; i++ {
		fieldName := policyType.Field(i).Name
		if strings.HasPrefix(fieldName, "SORT") {
			s.Contains(expectedSORTFields, fieldName, "Found unexpected SORT field %s, SORT fields must be cleared in exported policies in removeInternal()", fieldName)
		}
	}
}

func (s *PolicyServiceTestSuite) TestDryRunRuntime() {
	ctx := context.Background()
	runtimePolicy := storage.Policy_builder{
		Id:              "1",
		Name:            "RuntimePolicy",
		Severity:        storage.Severity_LOW_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		Categories:      []string{"test"},
		PolicySections: []*storage.PolicySection{
			storage.PolicySection_builder{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					storage.PolicyGroup_builder{
						FieldName: fieldnames.ProcessName,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "apt-get",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: fieldnames.PrivilegedContainer,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "true",
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		},
		EventSource:   storage.EventSource_DEPLOYMENT_EVENT,
		PolicyVersion: policyversion.CurrentVersion().String(),
	}.Build()
	resp, err := s.tested.DryRunPolicy(ctx, runtimePolicy)
	s.Nil(err)
	s.Nil(resp.GetAlerts())
}

func (s *PolicyServiceTestSuite) TestListPoliciesHandlesQueryAndPagination() {
	ctx := context.Background()
	basePolicy := fixtures.GetPolicy()
	policies := make([]*storage.Policy, 4)
	for i := 0; i < 4; i++ {
		p := basePolicy.CloneVT()
		p.SetId(fmt.Sprintf("policy-%d", i))
		policies = append(policies, p)
	}
	listPolicies := convertPoliciesToListPolicies(policies)
	policyDisabledBaseQuery := &v1.Query_BaseQuery{
		BaseQuery: v1.BaseQuery_builder{
			MatchFieldQuery: v1.MatchFieldQuery_builder{Field: search.Disabled.String(), Value: "false"}.Build(),
		}.Build(),
	}

	cases := []struct {
		name          string
		request       *v1.RawQuery
		expectedQuery *v1.Query
	}{
		{
			name:          "Empty query, get all policies",
			request:       &v1.RawQuery{},
			expectedQuery: v1.Query_builder{Pagination: v1.QueryPagination_builder{Limit: maxPoliciesReturned, Offset: 0}.Build()}.Build(),
		},
		{
			name:          "Empty query, paginate",
			request:       v1.RawQuery_builder{Pagination: v1.Pagination_builder{Limit: 2}.Build()}.Build(),
			expectedQuery: v1.Query_builder{Pagination: v1.QueryPagination_builder{Limit: 2, Offset: 0}.Build()}.Build(),
		},
		{
			name:          "Empty query, paginate and offset",
			request:       v1.RawQuery_builder{Pagination: v1.Pagination_builder{Limit: 20, Offset: 4}.Build()}.Build(),
			expectedQuery: v1.Query_builder{Pagination: v1.QueryPagination_builder{Limit: 20, Offset: 4}.Build()}.Build(),
		},
		{
			name:    "Non-empty query gets parsed properly",
			request: v1.RawQuery_builder{Query: search.NewQueryBuilder().AddBools(search.Disabled, false).Query()}.Build(),
			expectedQuery: v1.Query_builder{
				BaseQuery:  proto.ValueOrDefault(policyDisabledBaseQuery.BaseQuery),
				Pagination: v1.QueryPagination_builder{Limit: maxPoliciesReturned, Offset: 0}.Build(),
			}.Build(),
		},
		{
			name: "Non-empty query gets parsed properly, paginate",
			request: v1.RawQuery_builder{
				Query:      search.NewQueryBuilder().AddBools(search.Disabled, false).Query(),
				Pagination: v1.Pagination_builder{Limit: 2}.Build(),
			}.Build(),
			expectedQuery: v1.Query_builder{
				BaseQuery:  proto.ValueOrDefault(policyDisabledBaseQuery.BaseQuery),
				Pagination: v1.QueryPagination_builder{Limit: 2, Offset: 0}.Build(),
			}.Build(),
		},
		{
			name: "Non-empty query gets parsed properly, limit pagination to max and offset",
			request: v1.RawQuery_builder{
				Query:      search.NewQueryBuilder().AddBools(search.Disabled, false).Query(),
				Pagination: v1.Pagination_builder{Limit: 2000, Offset: 50}.Build(),
			}.Build(),
			expectedQuery: v1.Query_builder{
				BaseQuery:  proto.ValueOrDefault(policyDisabledBaseQuery.BaseQuery),
				Pagination: v1.QueryPagination_builder{Limit: 1000, Offset: 50}.Build(),
			}.Build(),
		},
	}
	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {
			s.policies.EXPECT().SearchRawPolicies(ctx, c.expectedQuery).Return(policies, nil).Times(1)
			resp, err := s.tested.ListPolicies(ctx, c.request)
			s.NoError(err)
			s.NotNil(resp)
			protoassert.SlicesEqual(s.T(), listPolicies, resp.GetPolicies())
		})
	}
}

func (s *PolicyServiceTestSuite) TestImportPolicy() {
	mockID := "1"
	mockName := "current version policy"
	mockSeverity := storage.Severity_LOW_SEVERITY
	mockLCStages := []storage.LifecycleStage{storage.LifecycleStage_RUNTIME}
	mockCategories := []string{"test"}
	importedPolicy := storage.Policy_builder{
		Id:              mockID,
		Name:            mockName,
		Severity:        mockSeverity,
		LifecycleStages: mockLCStages,
		Categories:      mockCategories,
		PolicySections: []*storage.PolicySection{
			storage.PolicySection_builder{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					storage.PolicyGroup_builder{
						FieldName: fieldnames.ProcessName,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "apt-get",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: fieldnames.PrivilegedContainer,
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "true",
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		},
		PolicyVersion: policyversion.CurrentVersion().String(),
		EventSource:   storage.EventSource_DEPLOYMENT_EVENT,
	}.Build()

	ctx := context.Background()
	ipr := &v1.ImportPolicyResponse{}
	ipr.SetSucceeded(true)
	ipr.SetPolicy(importedPolicy)
	ipr.SetErrors(nil)
	mockImportResp := []*v1.ImportPolicyResponse{
		ipr,
	}

	s.policies.EXPECT().ImportPolicies(ctx, []*storage.Policy{importedPolicy}, false).Return(mockImportResp, true, nil)
	s.mockLifecycleManager.EXPECT().UpsertPolicy(importedPolicy).Return(nil)
	s.policies.EXPECT().GetAllPolicies(gomock.Any()).Return(nil, nil)
	s.mockConnectionManager.EXPECT().PreparePoliciesAndBroadcast(gomock.Any())
	ipr2 := &v1.ImportPoliciesRequest{}
	ipr2.SetPolicies([]*storage.Policy{importedPolicy})
	resp, err := s.tested.ImportPolicies(ctx, ipr2)
	s.NoError(err)
	s.True(resp.GetAllSucceeded())
	s.Require().Len(resp.GetResponses(), 1)
	policyResp := resp.GetResponses()[0]
	resultPolicy := policyResp.GetPolicy()
	protoassert.SlicesEqual(s.T(), importedPolicy.GetPolicySections(), resultPolicy.GetPolicySections())
}

func (s *PolicyServiceTestSuite) testScopes(query string, mockClusters []*storage.Cluster, expectedScopes ...*storage.Scope) {
	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{}
	request.SetSearchParams(query)
	s.clusters.EXPECT().GetClusters(ctx).Return(mockClusters, nil).AnyTimes()
	response, err := s.tested.PolicyFromSearch(ctx, request)
	s.NoError(err)
	s.Empty(response.GetAlteredSearchTerms())
	s.False(response.GetHasNestedFields())
	s.NotNil(response.GetPolicy())
	protoassert.ElementsMatch(s.T(), expectedScopes, response.GetPolicy().GetScope())
}

func (s *PolicyServiceTestSuite) testMalformedScope(query string) {
	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{}
	request.SetSearchParams(query)
	_, err := s.tested.PolicyFromSearch(ctx, request)
	s.Error(err)
}

func (s *PolicyServiceTestSuite) testLifecycles(query string, expectedLifecycles ...storage.LifecycleStage) {
	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{}
	request.SetSearchParams(query)
	response, err := s.tested.PolicyFromSearch(ctx, request)
	s.NoError(err)
	s.Empty(response.GetAlteredSearchTerms())
	s.False(response.GetHasNestedFields())
	s.NotNil(response.GetPolicy())
	s.ElementsMatch(expectedLifecycles, response.GetPolicy().GetLifecycleStages())
}

func (s *PolicyServiceTestSuite) testPolicyGroups(query string, expectedPolicyGroups ...*storage.PolicyGroup) {
	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{}
	request.SetSearchParams(query)
	response, err := s.tested.PolicyFromSearch(ctx, request)
	s.NoError(err)
	s.Empty(response.GetAlteredSearchTerms())
	s.False(response.GetHasNestedFields())
	s.NotNil(response.GetPolicy())
	s.Require().Len(response.GetPolicy().GetPolicySections(), 1)
	policyGroups := response.GetPolicy().GetPolicySections()[0].GetPolicyGroups()
	protoassert.ElementsMatch(s.T(), expectedPolicyGroups, policyGroups)

	// These tests do not explicitly expect scopes so we should ensure that there are not scopes
	s.Nil(response.GetPolicy().GetScope())
}

func (s *PolicyServiceTestSuite) TestMalformedScopes() {
	queryString := "Deployment Label:"
	s.testMalformedScope(queryString)

	queryString = "Cluster:"
	s.testMalformedScope(queryString)

	queryString = "Namespace:"
	s.testMalformedScope(queryString)
}

func (s *PolicyServiceTestSuite) TestScopeWithMalformedLabel() {
	expectedScope := &storage.Scope{}
	expectedScope.SetNamespace("blah")
	queryString := "Deployment Label:+Namespace:blah"
	s.testScopes(queryString, nil, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeWithMalformedNamespace() {
	sl := &storage.Scope_Label{}
	sl.SetKey("blah")
	sl.SetValue("blah")
	expectedScope := &storage.Scope{}
	expectedScope.SetLabel(sl)
	queryString := "Deployment Label:blah=blah+Namespace:"
	s.testScopes(queryString, nil, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeWithMalformedCluster() {
	sl := &storage.Scope_Label{}
	sl.SetKey("blah")
	sl.SetValue("blah")
	expectedScope := &storage.Scope{}
	expectedScope.SetLabel(sl)
	queryString := "Deployment Label:blah=blah+Cluster:"
	s.testScopes(queryString, nil, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScope() {
	sl := &storage.Scope_Label{}
	sl.SetKey("app")
	sl.SetValue("collector")
	expectedScope := &storage.Scope{}
	expectedScope.SetCluster("remoteID")
	expectedScope.SetNamespace("stackrox")
	expectedScope.SetLabel(sl)
	cluster := &storage.Cluster{}
	cluster.SetName("remote")
	cluster.SetId("remoteID")
	mockClusters := []*storage.Cluster{
		cluster,
	}
	queryString := "Deployment Label:app=collector+Cluster:remote,+Namespace:stackrox"
	s.testScopes(queryString, mockClusters, expectedScope)
}

func (s *PolicyServiceTestSuite) TestManyScopes() {
	expectedScopes := []*storage.Scope{
		storage.Scope_builder{
			Cluster:   "remoteID",
			Namespace: "stackrox",
			Label: storage.Scope_Label_builder{
				Key:   "app",
				Value: "collector",
			}.Build(),
		}.Build(),
		storage.Scope_builder{
			Cluster:   "remoteID",
			Namespace: "hoops",
			Label: storage.Scope_Label_builder{
				Key:   "app",
				Value: "collector",
			}.Build(),
		}.Build(),
		storage.Scope_builder{
			Cluster:   "marsID",
			Namespace: "stackrox",
			Label: storage.Scope_Label_builder{
				Key:   "app",
				Value: "collector",
			}.Build(),
		}.Build(),
		storage.Scope_builder{
			Cluster:   "marsID",
			Namespace: "hoops",
			Label: storage.Scope_Label_builder{
				Key:   "app",
				Value: "collector",
			}.Build(),
		}.Build(),
		storage.Scope_builder{
			Cluster:   "remoteID",
			Namespace: "stackrox",
			Label: storage.Scope_Label_builder{
				Key:   "dunk",
				Value: "buckets",
			}.Build(),
		}.Build(),
		storage.Scope_builder{
			Cluster:   "remoteID",
			Namespace: "hoops",
			Label: storage.Scope_Label_builder{
				Key:   "dunk",
				Value: "buckets",
			}.Build(),
		}.Build(),
		storage.Scope_builder{
			Cluster:   "marsID",
			Namespace: "stackrox",
			Label: storage.Scope_Label_builder{
				Key:   "dunk",
				Value: "buckets",
			}.Build(),
		}.Build(),
		storage.Scope_builder{
			Cluster:   "marsID",
			Namespace: "hoops",
			Label: storage.Scope_Label_builder{
				Key:   "dunk",
				Value: "buckets",
			}.Build(),
		}.Build(),
	}
	cluster := &storage.Cluster{}
	cluster.SetName("mars")
	cluster.SetId("marsID")
	cluster2 := &storage.Cluster{}
	cluster2.SetName("remote")
	cluster2.SetId("remoteID")
	mockClusters := []*storage.Cluster{
		cluster,
		cluster2,
	}
	queryString := "Deployment Label:app=collector,dunk=buckets+Cluster:remote,mars,+Namespace:stackrox,hoops"
	s.testScopes(queryString, mockClusters, expectedScopes...)
}

func (s *PolicyServiceTestSuite) TestScopeOnlyCluster() {
	expectedScope := &storage.Scope{}
	expectedScope.SetCluster("remoteID")
	cluster := &storage.Cluster{}
	cluster.SetName("remote")
	cluster.SetId("remoteID")
	mockClusters := []*storage.Cluster{
		cluster,
	}
	queryString := "Cluster:remote"
	s.testScopes(queryString, mockClusters, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeOnlyNamespace() {
	expectedScope := &storage.Scope{}
	expectedScope.SetNamespace("Joseph Rules")
	queryString := "Namespace:Joseph Rules"
	s.testScopes(queryString, nil, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeOnlyLabel() {
	sl := &storage.Scope_Label{}
	sl.SetKey("Joseph")
	sl.SetValue("Rules")
	expectedScope := &storage.Scope{}
	expectedScope.SetLabel(sl)
	queryString := "Deployment Label:Joseph=Rules"
	s.testScopes(queryString, nil, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeOddLabelFormats() {
	expectedScopes := []*storage.Scope{
		storage.Scope_builder{
			Label: storage.Scope_Label_builder{
				Key: "Joseph",
			}.Build(),
		}.Build(),
		storage.Scope_builder{
			Label: storage.Scope_Label_builder{
				Key:   "a",
				Value: "b=c",
			}.Build(),
		}.Build(),
	}
	queryString := "Deployment Label:Joseph,a=b=c"
	s.testScopes(queryString, nil, expectedScopes...)
}

func (s *PolicyServiceTestSuite) TestScopeClusterNamespace() {
	expectedScope := &storage.Scope{}
	expectedScope.SetCluster("remoteID")
	expectedScope.SetNamespace("stackrox")
	cluster := &storage.Cluster{}
	cluster.SetName("remote")
	cluster.SetId("remoteID")
	mockClusters := []*storage.Cluster{
		cluster,
	}
	queryString := "Cluster:remote,+Namespace:stackrox"
	s.testScopes(queryString, mockClusters, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeClusterLabel() {
	sl := &storage.Scope_Label{}
	sl.SetKey("Joseph")
	sl.SetValue("Rules")
	expectedScope := &storage.Scope{}
	expectedScope.SetCluster("remoteID")
	expectedScope.SetLabel(sl)
	cluster := &storage.Cluster{}
	cluster.SetName("remote")
	cluster.SetId("remoteID")
	mockClusters := []*storage.Cluster{
		cluster,
	}
	queryString := "Cluster:remote,+Deployment Label:Joseph=Rules"
	s.testScopes(queryString, mockClusters, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeNamespaceLabel() {
	sl := &storage.Scope_Label{}
	sl.SetKey("Joseph")
	sl.SetValue("Rules")
	expectedScope := &storage.Scope{}
	expectedScope.SetNamespace("stackrox")
	expectedScope.SetLabel(sl)
	queryString := "Namespace:stackrox+Deployment Label:Joseph=Rules"
	s.testScopes(queryString, nil, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeClusterRegex() {
	scope := &storage.Scope{}
	scope.SetCluster("remoteID")
	scope2 := &storage.Scope{}
	scope2.SetCluster("aDifferentCluster")
	expectedScopes := []*storage.Scope{
		scope,
		scope2,
	}
	cluster := &storage.Cluster{}
	cluster.SetName("remote")
	cluster.SetId("remoteID")
	cluster2 := &storage.Cluster{}
	cluster2.SetName("remSomeOtherCluster")
	cluster2.SetId("aDifferentCluster")
	cluster3 := &storage.Cluster{}
	cluster3.SetName("dontReturnThisCluster")
	cluster3.SetId("super good ID")
	mockClusters := []*storage.Cluster{
		cluster,
		cluster2,
		cluster3,
	}
	queryString := "Cluster:r/rem.*"
	s.testScopes(queryString, mockClusters, expectedScopes...)
}

func (s *PolicyServiceTestSuite) TestRuntimeLifecycle() {
	s.testLifecycles("CVE:abcd+Process Name:123", storage.LifecycleStage_RUNTIME)
	// Note that kube events/audit logs fields are simply dropped instead of being returned as unconvertableFields, since they are not searchable.
	s.testLifecycles("CVE:abcd+Kubernetes Resource:PODS_EXEC", storage.LifecycleStage_BUILD, storage.LifecycleStage_DEPLOY)
	s.testLifecycles("CVE:abcd+Kubernetes Resource:PODS_EXEC+Is Impersonated User:true", storage.LifecycleStage_BUILD, storage.LifecycleStage_DEPLOY)
}

func (s *PolicyServiceTestSuite) TestBuildAndDeployLifecycles() {
	s.testLifecycles("CVE:abcd", storage.LifecycleStage_DEPLOY, storage.LifecycleStage_BUILD)
}

func (s *PolicyServiceTestSuite) TestOnePolicyField() {
	pv := &storage.PolicyValue{}
	pv.SetValue("abcd")
	expectedPolicyGroup := &storage.PolicyGroup{}
	expectedPolicyGroup.SetFieldName(fieldnames.CVE)
	expectedPolicyGroup.SetBooleanOperator(storage.BooleanOperator_OR)
	expectedPolicyGroup.SetValues([]*storage.PolicyValue{
		pv,
	})
	queryString := "CVE:abcd"
	s.testPolicyGroups(queryString, expectedPolicyGroup)
}

func (s *PolicyServiceTestSuite) TestMultiplePolicyFields() {
	expectedPolicyGroups := []*storage.PolicyGroup{
		storage.PolicyGroup_builder{
			FieldName:       fieldnames.CVE,
			BooleanOperator: storage.BooleanOperator_OR,
			Values: []*storage.PolicyValue{
				storage.PolicyValue_builder{
					Value: "abcd",
				}.Build(),
			},
		}.Build(),
		storage.PolicyGroup_builder{
			FieldName:       fieldnames.ImageComponent,
			BooleanOperator: storage.BooleanOperator_OR,
			Values: []*storage.PolicyValue{
				storage.PolicyValue_builder{
					Value: "ewjnv=",
				}.Build(),
			},
		}.Build(),
	}
	queryString := "CVE:abcd+Component:ewjnv"
	s.testPolicyGroups(queryString, expectedPolicyGroups...)
}

func (s *PolicyServiceTestSuite) TestUnconvertableFields() {
	expectedPolicyGroup := []*storage.PolicyGroup{
		storage.PolicyGroup_builder{
			FieldName:       fieldnames.CVE,
			BooleanOperator: storage.BooleanOperator_OR,
			Values: []*storage.PolicyValue{
				storage.PolicyValue_builder{
					Value: "abcd",
				}.Build(),
			},
		}.Build(),
	}
	expectedUnconvertable := []string{search.CVESuppressed.String()}

	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{}
	request.SetSearchParams("CVE:abcd+CVE Snoozed:hrkrj")
	response, err := s.tested.PolicyFromSearch(ctx, request)
	s.NoError(err)
	s.Equal(expectedUnconvertable, response.GetAlteredSearchTerms())
	s.False(response.GetHasNestedFields())
	s.NotNil(response.GetPolicy())
	s.Require().Len(response.GetPolicy().GetPolicySections(), 1)
	policyGroups := response.GetPolicy().GetPolicySections()[0].GetPolicyGroups()
	protoassert.ElementsMatch(s.T(), expectedPolicyGroup, policyGroups)
}

func (s *PolicyServiceTestSuite) TestNoConvertableFields() {
	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{}
	request.SetSearchParams("Deployment:abcd+CVE Snoozed:hrkrj+Deployment Label:+NotASearchTerm:jkjksdr")
	_, err := s.tested.PolicyFromSearch(ctx, request)
	s.Error(err)
	s.Contains(err.Error(), "no valid policy groups or scopes")
}

func (s *PolicyServiceTestSuite) TestMakePolicyWithCombinations() {
	expectedPolicyGroups := []*storage.PolicyGroup{
		storage.PolicyGroup_builder{
			FieldName: fieldnames.EnvironmentVariable,
			Values: []*storage.PolicyValue{
				storage.PolicyValue_builder{
					Value: "v=z=x",
				}.Build(),
				storage.PolicyValue_builder{
					Value: "v=z=w",
				}.Build(),
				storage.PolicyValue_builder{
					Value: "v=y=x",
				}.Build(),
				storage.PolicyValue_builder{
					Value: "v=y=w",
				}.Build(),
			},
		}.Build(),
		storage.PolicyGroup_builder{
			FieldName: fieldnames.DockerfileLine,
			Values: []*storage.PolicyValue{
				storage.PolicyValue_builder{
					Value: "a=",
				}.Build(),
				storage.PolicyValue_builder{
					Value: "b=",
				}.Build(),
			},
		}.Build(),
	}

	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{}
	request.SetSearchParams("Dockerfile Instruction Keyword:a,b+Environment Key:z,y+Environment Value:x,w+Environment Variable Source:v")
	response, err := s.tested.PolicyFromSearch(ctx, request)
	s.NoError(err)
	s.False(response.GetHasNestedFields())
	s.Empty(response.GetAlteredSearchTerms())
	s.Len(response.GetPolicy().GetPolicySections(), 1)
	protoassert.ElementsMatch(s.T(), expectedPolicyGroups, response.GetPolicy().GetPolicySections()[0].GetPolicyGroups())
}

func (s *PolicyServiceTestSuite) TestEnvironmentXLifecycle() {
	expectedPolicyGroup := []*storage.PolicyGroup{
		storage.PolicyGroup_builder{
			FieldName: fieldnames.EnvironmentVariable,
			Values: []*storage.PolicyValue{
				storage.PolicyValue_builder{
					Value: "=z=",
				}.Build(),
			},
		}.Build(),
	}

	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{}
	request.SetSearchParams("Environment Key:z")
	response, err := s.tested.PolicyFromSearch(ctx, request)
	s.NoError(err)
	s.False(response.GetHasNestedFields())
	s.Empty(response.GetAlteredSearchTerms())
	protoassert.ElementsMatch(s.T(), expectedPolicyGroup, response.GetPolicy().GetPolicySections()[0].GetPolicyGroups())
	expectedLifecycleStages := []storage.LifecycleStage{storage.LifecycleStage_DEPLOY}
	s.ElementsMatch(expectedLifecycleStages, response.GetPolicy().GetLifecycleStages())
}

// This test is the expected behavior after the sample mitre data injection is removed.
func (s *PolicyServiceTestSuite) TestMitreVectors() {
	s.policies.EXPECT().GetPolicy(gomock.Any(), "policy1").Return(storage.Policy_builder{
		Id: "policy1",
		MitreAttackVectors: []*storage.Policy_MitreAttackVectors{
			storage.Policy_MitreAttackVectors_builder{
				Tactic:     "tactic1",
				Techniques: []string{"tech1"},
			}.Build(),
			storage.Policy_MitreAttackVectors_builder{
				Tactic:     "tactic2",
				Techniques: []string{"tech2"},
			}.Build(),
		},
	}.Build(), true, nil)

	s.mitreVectorStore.EXPECT().Get("tactic1").Return(
		getFakeVector("tactic1", "tech1", "tech2", "tech3"), nil,
	)
	s.mitreVectorStore.EXPECT().Get("tactic2").Return(
		getFakeVector("tactic2", "tech1", "tech2"), nil,
	)

	gpmvr := &v1.GetPolicyMitreVectorsRequest{}
	gpmvr.SetId("policy1")
	response, err := s.tested.GetPolicyMitreVectors(context.Background(), gpmvr)
	s.NoError(err)
	protoassert.ElementsMatch(s.T(), []*storage.MitreAttackVector{
		getFakeVector("tactic1", "tech1"),
		getFakeVector("tactic2", "tech2"),
	}, response.GetVectors())
}

func getFakeVector(tactic string, techniques ...string) *storage.MitreAttackVector {
	mt := &storage.MitreTactic{}
	mt.SetId(tactic)
	mt.SetName(tactic)
	mt.SetDescription(tactic)
	resp := &storage.MitreAttackVector{}
	resp.SetTactic(mt)

	for _, technique := range techniques {
		mt2 := &storage.MitreTechnique{}
		mt2.SetId(technique)
		mt2.SetName(technique)
		mt2.SetDescription(technique)
		resp.SetTechniques(append(resp.GetTechniques(), mt2))
	}

	return resp
}

func (s *PolicyServiceTestSuite) TestDeletingDefaultPolicyIsBlocked() {
	ctx := context.Background()

	// arrange
	mockPolicy := &storage.Policy{}
	mockPolicy.SetId(mockRequestOneID.GetPolicyIds()[0])
	mockPolicy.SetIsDefault(true)
	s.policies.EXPECT().GetPolicy(ctx, mockPolicy.GetId()).Return(mockPolicy, true, nil)
	expectedErr := errors.Wrap(errox.InvalidArgs, "A default policy cannot be deleted. (You can disable a default policy, but not delete it.)")

	// act
	fakeResourceByIDRequest := &v1.ResourceByID{}
	fakeResourceByIDRequest.SetId(mockPolicy.GetId())
	resp, err := s.tested.DeletePolicy(ctx, fakeResourceByIDRequest)

	// assert
	s.Require().Error(err, expectedErr)
	s.Require().Nil(resp)
}

func (s *PolicyServiceTestSuite) TestDeletingNonExistentPolicyDoesNothing() {
	ctx := context.Background()

	// arrange
	mockPolicyID := mockRequestOneID.GetPolicyIds()[0] // used only for the ID
	s.policies.EXPECT().GetPolicy(ctx, mockPolicyID).Return(nil, false, nil)

	// act
	fakeResourceByIDRequest := &v1.ResourceByID{}
	fakeResourceByIDRequest.SetId(mockPolicyID)
	resp, err := s.tested.DeletePolicy(ctx, fakeResourceByIDRequest)

	// assert
	s.NoError(err)
	s.Empty(resp)
}

func (s *PolicyServiceTestSuite) TestDeletingPolicyErrOnDbError() {
	ctx := context.Background()

	// arrange
	mockPolicyID := mockRequestOneID.GetPolicyIds()[0] // used only for the ID
	dbErr := errors.New("the deebee has failed you")
	s.policies.EXPECT().GetPolicy(ctx, mockPolicyID).Return(nil, true, dbErr)
	expectedErr := errors.Wrap(dbErr, "DB error while trying to delete policy")

	// act
	fakeResourceByIDRequest := &v1.ResourceByID{}
	fakeResourceByIDRequest.SetId(mockPolicyID)
	resp, err := s.tested.DeletePolicy(ctx, fakeResourceByIDRequest)

	// assert
	s.Require().Error(err, expectedErr)
	s.Require().Nil(resp)
}
