package service

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	lifecycleMocks "github.com/stackrox/rox/central/detection/lifecycle/mocks"
	mitreMocks "github.com/stackrox/rox/central/mitre/datastore/mocks"
	"github.com/stackrox/rox/central/policy/datastore/mocks"
	connectionMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	detectionMocks "github.com/stackrox/rox/pkg/detection/mocks"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/status"
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

type testDeploymentMatcher struct {
	*detectionMocks.MockPolicySet
}

func (t *testDeploymentMatcher) RemoveNotifier(_ string) error {
	return nil
}

type PolicyServiceTestSuite struct {
	suite.Suite
	policies              *mocks.MockDataStore
	clusters              *clusterMocks.MockDataStore
	mitreVectorStore      *mitreMocks.MockMitreAttackReadOnlyDataStore
	mockBuildTimePolicies *detectionMocks.MockPolicySet
	mockLifecycleManager  *lifecycleMocks.MockManager
	mockConnectionManager *connectionMocks.MockManager
	tested                Service

	envIsolator *envisolator.EnvIsolator

	mockCtrl *gomock.Controller
}

func (s *PolicyServiceTestSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())

	s.mockCtrl = gomock.NewController(s.T())

	s.policies = mocks.NewMockDataStore(s.mockCtrl)

	s.clusters = clusterMocks.NewMockDataStore(s.mockCtrl)

	s.mockBuildTimePolicies = detectionMocks.NewMockPolicySet(s.mockCtrl)
	s.mockLifecycleManager = lifecycleMocks.NewMockManager(s.mockCtrl)
	s.mockConnectionManager = connectionMocks.NewMockManager(s.mockCtrl)
	s.mitreVectorStore = mitreMocks.NewMockMitreAttackReadOnlyDataStore(s.mockCtrl)

	s.tested = New(
		s.policies,
		s.clusters,
		nil,
		nil,
		nil,
		s.mitreVectorStore,
		nil,
		&testDeploymentMatcher{s.mockBuildTimePolicies},
		s.mockLifecycleManager,
		nil,
		nil,
		s.mockConnectionManager,
	)
}

func (s *PolicyServiceTestSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
	s.mockCtrl.Finish()
}

func (s *PolicyServiceTestSuite) compareErrorsToExpected(expectedErrors []*v1.ExportPolicyError, apiError error) {
	apiStatus, ok := status.FromError(apiError)
	s.Require().True(ok)
	details := apiStatus.Details()
	s.Len(details, 1)
	exportErrors, ok := details[0].(*v1.ExportPoliciesErrorList)
	s.Require().True(ok)
	// actual errors == expected errors ignoring order
	s.Len(exportErrors.GetErrors(), len(expectedErrors))
	for _, expected := range expectedErrors {
		s.Contains(exportErrors.GetErrors(), expected)
	}
}

func makeError(errorID, errorString string) *v1.ExportPolicyError {
	return &v1.ExportPolicyError{
		PolicyId: errorID,
		Error: &v1.PolicyError{
			Error: errorString,
		},
	}
}

func (s *PolicyServiceTestSuite) TestExportInvalidIDFails() {
	ctx := context.Background()
	mockErrors := []*v1.ExportPolicyError{
		makeError(mockRequestOneID.PolicyIds[0], "not found"),
	}
	s.policies.EXPECT().GetPolicies(ctx, mockRequestOneID.PolicyIds).Return(make([]*storage.Policy, 0), []int{0}, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestOneID)
	s.Nil(resp)
	s.Error(err)
	s.compareErrorsToExpected(mockErrors, err)
}

func (s *PolicyServiceTestSuite) TestExportValidIDSucceeds() {
	ctx := context.Background()
	mockPolicy := &storage.Policy{
		Id: mockRequestOneID.PolicyIds[0],
	}
	s.policies.EXPECT().GetPolicies(ctx, mockRequestOneID.PolicyIds).Return([]*storage.Policy{mockPolicy}, nil, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestOneID)
	s.NoError(err)
	s.NotNil(resp)
	s.Len(resp.GetPolicies(), 1)
	s.Equal(mockPolicy, resp.Policies[0])
}

func (s *PolicyServiceTestSuite) TestExportMixedSuccessAndMissing() {
	ctx := context.Background()
	mockPolicy := &storage.Policy{
		Id: mockRequestTwoIDs.PolicyIds[0],
	}
	mockErrors := []*v1.ExportPolicyError{
		makeError(mockRequestTwoIDs.PolicyIds[1], "not found"),
	}
	s.policies.EXPECT().GetPolicies(ctx, mockRequestTwoIDs.PolicyIds).Return([]*storage.Policy{mockPolicy}, []int{1}, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestTwoIDs)
	s.Nil(resp)
	s.Error(err)
	s.compareErrorsToExpected(mockErrors, err)
}

func (s *PolicyServiceTestSuite) TestExportMultipleFailures() {
	ctx := context.Background()
	mockErrors := []*v1.ExportPolicyError{
		makeError(mockRequestTwoIDs.PolicyIds[0], "not found"),
		makeError(mockRequestTwoIDs.PolicyIds[1], "not found"),
	}
	s.policies.EXPECT().GetPolicies(ctx, mockRequestTwoIDs.PolicyIds).Return(make([]*storage.Policy, 0), []int{0, 1}, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestTwoIDs)
	s.Nil(resp)
	s.Error(err)
	s.compareErrorsToExpected(mockErrors, err)
}

func (s *PolicyServiceTestSuite) TestExportedPolicyHasNoSortFields() {
	ctx := context.Background()
	mockPolicy := &storage.Policy{
		Id:                 mockRequestOneID.PolicyIds[0],
		SORTName:           "abc",
		SORTLifecycleStage: "def",
	}
	expectedPolicy := &storage.Policy{
		Id: mockRequestOneID.PolicyIds[0],
	}
	s.policies.EXPECT().GetPolicies(ctx, mockRequestOneID.PolicyIds).Return([]*storage.Policy{mockPolicy}, nil, nil)
	resp, err := s.tested.ExportPolicies(ctx, mockRequestOneID)
	s.NoError(err)
	s.NotNil(resp)
	s.Len(resp.GetPolicies(), 1)
	s.Equal(expectedPolicy, resp.Policies[0])
}

func (s *PolicyServiceTestSuite) TestPoliciesHaveNoUnexpectedSORTFields() {
	expectedSORTFields := set.NewStringSet("SORTLifecycleStage", "SORTEnforcement", "SORTName")
	var policy storage.Policy
	policyType := reflect.TypeOf(policy)
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
	runtimePolicy := &storage.Policy{
		Id:              "1",
		Name:            "RuntimePolicy",
		Severity:        storage.Severity_LOW_SEVERITY,
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		Categories:      []string{"test"},
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.ProcessName,
						Values: []*storage.PolicyValue{
							{
								Value: "apt-get",
							},
						},
					},
					{
						FieldName: fieldnames.PrivilegedContainer,
						Values: []*storage.PolicyValue{
							{
								Value: "true",
							},
						},
					},
				},
			},
		},
		EventSource:   storage.EventSource_DEPLOYMENT_EVENT,
		PolicyVersion: policyversion.CurrentVersion().String(),
	}
	resp, err := s.tested.DryRunPolicy(ctx, runtimePolicy)
	s.Nil(err)
	s.Nil(resp.GetAlerts())
}

func (s *PolicyServiceTestSuite) TestImportPolicy() {
	mockID := "1"
	mockName := "current version policy"
	mockSeverity := storage.Severity_LOW_SEVERITY
	mockLCStages := []storage.LifecycleStage{storage.LifecycleStage_RUNTIME}
	mockCategories := []string{"test"}
	importedPolicy := &storage.Policy{
		Id:              mockID,
		Name:            mockName,
		Severity:        mockSeverity,
		LifecycleStages: mockLCStages,
		Categories:      mockCategories,
		PolicySections: []*storage.PolicySection{
			{
				SectionName: "section-1",
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: fieldnames.ProcessName,
						Values: []*storage.PolicyValue{
							{
								Value: "apt-get",
							},
						},
					},
					{
						FieldName: fieldnames.PrivilegedContainer,
						Values: []*storage.PolicyValue{
							{
								Value: "true",
							},
						},
					},
				},
			},
		},
		PolicyVersion: policyversion.CurrentVersion().String(),
		EventSource:   storage.EventSource_DEPLOYMENT_EVENT,
	}

	ctx := context.Background()
	mockImportResp := []*v1.ImportPolicyResponse{
		{
			Succeeded: true,
			Policy:    importedPolicy,
			Errors:    nil,
		},
	}

	s.policies.EXPECT().ImportPolicies(ctx, []*storage.Policy{importedPolicy}, false).Return(mockImportResp, true, nil)
	s.mockBuildTimePolicies.EXPECT().RemovePolicy(importedPolicy.GetId())
	s.mockLifecycleManager.EXPECT().UpsertPolicy(importedPolicy).Return(nil)
	s.policies.EXPECT().GetAllPolicies(gomock.Any()).Return(nil, nil)
	s.mockConnectionManager.EXPECT().PreparePoliciesAndBroadcast(gomock.Any())
	resp, err := s.tested.ImportPolicies(ctx, &v1.ImportPoliciesRequest{
		Policies: []*storage.Policy{importedPolicy},
	})
	s.NoError(err)
	s.True(resp.AllSucceeded)
	s.Require().Len(resp.GetResponses(), 1)
	policyResp := resp.GetResponses()[0]
	resultPolicy := policyResp.GetPolicy()
	s.Equal(importedPolicy.GetPolicySections(), resultPolicy.GetPolicySections())
}

func (s *PolicyServiceTestSuite) testScopes(query string, mockClusters []*storage.Cluster, expectedScopes ...*storage.Scope) {
	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{
		SearchParams: query,
	}
	s.clusters.EXPECT().GetClusters(ctx).Return(mockClusters, nil).AnyTimes()
	response, err := s.tested.PolicyFromSearch(ctx, request)
	s.NoError(err)
	s.Empty(response.GetAlteredSearchTerms())
	s.False(response.GetHasNestedFields())
	s.NotNil(response.GetPolicy())
	s.ElementsMatch(expectedScopes, response.GetPolicy().GetScope())
}

func (s *PolicyServiceTestSuite) testMalformedScope(query string) {
	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{
		SearchParams: query,
	}
	_, err := s.tested.PolicyFromSearch(ctx, request)
	s.Error(err)
}

func (s *PolicyServiceTestSuite) testLifecycles(query string, expectedLifecycles ...storage.LifecycleStage) {
	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{
		SearchParams: query,
	}
	response, err := s.tested.PolicyFromSearch(ctx, request)
	s.NoError(err)
	s.Empty(response.GetAlteredSearchTerms())
	s.False(response.GetHasNestedFields())
	s.NotNil(response.GetPolicy())
	s.ElementsMatch(expectedLifecycles, response.GetPolicy().GetLifecycleStages())
}

func (s *PolicyServiceTestSuite) testPolicyGroups(query string, expectedPolicyGroups ...*storage.PolicyGroup) {
	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{
		SearchParams: query,
	}
	response, err := s.tested.PolicyFromSearch(ctx, request)
	s.NoError(err)
	s.Empty(response.GetAlteredSearchTerms())
	s.False(response.GetHasNestedFields())
	s.NotNil(response.GetPolicy())
	s.Require().Len(response.GetPolicy().GetPolicySections(), 1)
	policyGroups := response.GetPolicy().GetPolicySections()[0].GetPolicyGroups()
	s.ElementsMatch(expectedPolicyGroups, policyGroups)

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
	expectedScope := &storage.Scope{
		Namespace: "blah",
	}
	queryString := "Deployment Label:+Namespace:blah"
	s.testScopes(queryString, nil, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeWithMalformedNamespace() {
	expectedScope := &storage.Scope{
		Label: &storage.Scope_Label{
			Key:   "blah",
			Value: "blah",
		},
	}
	queryString := "Deployment Label:blah=blah+Namespace:"
	s.testScopes(queryString, nil, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeWithMalformedCluster() {
	expectedScope := &storage.Scope{
		Label: &storage.Scope_Label{
			Key:   "blah",
			Value: "blah",
		},
	}
	queryString := "Deployment Label:blah=blah+Cluster:"
	s.testScopes(queryString, nil, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScope() {
	expectedScope := &storage.Scope{
		Cluster:   "remoteID",
		Namespace: "stackrox",
		Label: &storage.Scope_Label{
			Key:   "app",
			Value: "collector",
		},
	}
	mockClusters := []*storage.Cluster{
		{
			Name: "remote",
			Id:   "remoteID",
		},
	}
	queryString := "Deployment Label:app=collector+Cluster:remote,+Namespace:stackrox"
	s.testScopes(queryString, mockClusters, expectedScope)
}

func (s *PolicyServiceTestSuite) TestManyScopes() {
	expectedScopes := []*storage.Scope{
		{
			Cluster:   "remoteID",
			Namespace: "stackrox",
			Label: &storage.Scope_Label{
				Key:   "app",
				Value: "collector",
			},
		},
		{
			Cluster:   "remoteID",
			Namespace: "hoops",
			Label: &storage.Scope_Label{
				Key:   "app",
				Value: "collector",
			},
		},
		{
			Cluster:   "marsID",
			Namespace: "stackrox",
			Label: &storage.Scope_Label{
				Key:   "app",
				Value: "collector",
			},
		},
		{
			Cluster:   "marsID",
			Namespace: "hoops",
			Label: &storage.Scope_Label{
				Key:   "app",
				Value: "collector",
			},
		},
		{
			Cluster:   "remoteID",
			Namespace: "stackrox",
			Label: &storage.Scope_Label{
				Key:   "dunk",
				Value: "buckets",
			},
		},
		{
			Cluster:   "remoteID",
			Namespace: "hoops",
			Label: &storage.Scope_Label{
				Key:   "dunk",
				Value: "buckets",
			},
		},
		{
			Cluster:   "marsID",
			Namespace: "stackrox",
			Label: &storage.Scope_Label{
				Key:   "dunk",
				Value: "buckets",
			},
		},
		{
			Cluster:   "marsID",
			Namespace: "hoops",
			Label: &storage.Scope_Label{
				Key:   "dunk",
				Value: "buckets",
			},
		},
	}
	mockClusters := []*storage.Cluster{
		{
			Name: "mars",
			Id:   "marsID",
		},
		{
			Name: "remote",
			Id:   "remoteID",
		},
	}
	queryString := "Deployment Label:app=collector,dunk=buckets+Cluster:remote,mars,+Namespace:stackrox,hoops"
	s.testScopes(queryString, mockClusters, expectedScopes...)
}

func (s *PolicyServiceTestSuite) TestScopeOnlyCluster() {
	expectedScope := &storage.Scope{
		Cluster: "remoteID",
	}
	mockClusters := []*storage.Cluster{
		{
			Name: "remote",
			Id:   "remoteID",
		},
	}
	queryString := "Cluster:remote"
	s.testScopes(queryString, mockClusters, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeOnlyNamespace() {
	expectedScope := &storage.Scope{
		Namespace: "Joseph Rules",
	}
	queryString := "Namespace:Joseph Rules"
	s.testScopes(queryString, nil, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeOnlyLabel() {
	expectedScope := &storage.Scope{
		Label: &storage.Scope_Label{
			Key:   "Joseph",
			Value: "Rules",
		},
	}
	queryString := "Deployment Label:Joseph=Rules"
	s.testScopes(queryString, nil, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeOddLabelFormats() {
	expectedScopes := []*storage.Scope{
		{
			Label: &storage.Scope_Label{
				Key: "Joseph",
			},
		},
		{
			Label: &storage.Scope_Label{
				Key:   "a",
				Value: "b=c",
			},
		},
	}
	queryString := "Deployment Label:Joseph,a=b=c"
	s.testScopes(queryString, nil, expectedScopes...)
}

func (s *PolicyServiceTestSuite) TestScopeClusterNamespace() {
	expectedScope := &storage.Scope{
		Cluster:   "remoteID",
		Namespace: "stackrox",
	}
	mockClusters := []*storage.Cluster{
		{
			Name: "remote",
			Id:   "remoteID",
		},
	}
	queryString := "Cluster:remote,+Namespace:stackrox"
	s.testScopes(queryString, mockClusters, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeClusterLabel() {
	expectedScope := &storage.Scope{
		Cluster: "remoteID",
		Label: &storage.Scope_Label{
			Key:   "Joseph",
			Value: "Rules",
		},
	}
	mockClusters := []*storage.Cluster{
		{
			Name: "remote",
			Id:   "remoteID",
		},
	}
	queryString := "Cluster:remote,+Deployment Label:Joseph=Rules"
	s.testScopes(queryString, mockClusters, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeNamespaceLabel() {
	expectedScope := &storage.Scope{
		Namespace: "stackrox",
		Label: &storage.Scope_Label{
			Key:   "Joseph",
			Value: "Rules",
		},
	}
	queryString := "Namespace:stackrox+Deployment Label:Joseph=Rules"
	s.testScopes(queryString, nil, expectedScope)
}

func (s *PolicyServiceTestSuite) TestScopeClusterRegex() {
	expectedScopes := []*storage.Scope{
		{
			Cluster: "remoteID",
		},
		{
			Cluster: "aDifferentCluster",
		},
	}
	mockClusters := []*storage.Cluster{
		{
			Name: "remote",
			Id:   "remoteID",
		},
		{
			Name: "remSomeOtherCluster",
			Id:   "aDifferentCluster",
		},
		{
			Name: "dontReturnThisCluster",
			Id:   "super good ID",
		},
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
	expectedPolicyGroup := &storage.PolicyGroup{
		FieldName:       fieldnames.CVE,
		BooleanOperator: storage.BooleanOperator_OR,
		Values: []*storage.PolicyValue{
			{
				Value: "abcd",
			},
		},
	}
	queryString := "CVE:abcd"
	s.testPolicyGroups(queryString, expectedPolicyGroup)
}

func (s *PolicyServiceTestSuite) TestMultiplePolicyFields() {
	expectedPolicyGroups := []*storage.PolicyGroup{
		{
			FieldName:       fieldnames.CVE,
			BooleanOperator: storage.BooleanOperator_OR,
			Values: []*storage.PolicyValue{
				{
					Value: "abcd",
				},
			},
		},
		{
			FieldName:       fieldnames.ImageComponent,
			BooleanOperator: storage.BooleanOperator_OR,
			Values: []*storage.PolicyValue{
				{
					Value: "ewjnv=",
				},
			},
		},
	}
	queryString := "CVE:abcd+Component:ewjnv"
	s.testPolicyGroups(queryString, expectedPolicyGroups...)
}

func (s *PolicyServiceTestSuite) TestUnconvertableFields() {
	expectedPolicyGroup := []*storage.PolicyGroup{
		{
			FieldName:       fieldnames.CVE,
			BooleanOperator: storage.BooleanOperator_OR,
			Values: []*storage.PolicyValue{
				{
					Value: "abcd",
				},
			},
		},
	}
	expectedUnconvertable := []string{search.CVESuppressed.String()}

	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{
		SearchParams: "CVE:abcd+CVE Snoozed:hrkrj",
	}
	response, err := s.tested.PolicyFromSearch(ctx, request)
	s.NoError(err)
	s.Equal(expectedUnconvertable, response.GetAlteredSearchTerms())
	s.False(response.GetHasNestedFields())
	s.NotNil(response.GetPolicy())
	s.Require().Len(response.GetPolicy().GetPolicySections(), 1)
	policyGroups := response.GetPolicy().GetPolicySections()[0].GetPolicyGroups()
	s.ElementsMatch(expectedPolicyGroup, policyGroups)
}

func (s *PolicyServiceTestSuite) TestNoConvertableFields() {
	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{
		SearchParams: "Deployment:abcd+CVE Snoozed:hrkrj+Deployment Label:+NotASearchTerm:jkjksdr",
	}
	_, err := s.tested.PolicyFromSearch(ctx, request)
	s.Error(err)
	s.Contains(err.Error(), "no valid policy groups or scopes")
}

func (s *PolicyServiceTestSuite) TestMakePolicyWithCombinations() {
	expectedPolicyGroups := []*storage.PolicyGroup{
		{
			FieldName: fieldnames.EnvironmentVariable,
			Values: []*storage.PolicyValue{
				{
					Value: "v=z=x",
				},
				{
					Value: "v=z=w",
				},
				{
					Value: "v=y=x",
				},
				{
					Value: "v=y=w",
				},
			},
		},
		{
			FieldName: fieldnames.DockerfileLine,
			Values: []*storage.PolicyValue{
				{
					Value: "a=",
				},
				{
					Value: "b=",
				},
			},
		},
	}

	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{
		SearchParams: "Dockerfile Instruction Keyword:a,b+Environment Key:z,y+Environment Value:x,w+Environment Variable Source:v",
	}
	response, err := s.tested.PolicyFromSearch(ctx, request)
	s.NoError(err)
	s.False(response.GetHasNestedFields())
	s.Empty(response.GetAlteredSearchTerms())
	s.Len(response.GetPolicy().GetPolicySections(), 1)
	s.ElementsMatch(expectedPolicyGroups, response.GetPolicy().GetPolicySections()[0].GetPolicyGroups())
}

func (s *PolicyServiceTestSuite) TestEnvironmentXLifecycle() {
	expectedPolicyGroup := []*storage.PolicyGroup{
		{
			FieldName: fieldnames.EnvironmentVariable,
			Values: []*storage.PolicyValue{
				{
					Value: "=z=",
				},
			},
		},
	}

	ctx := context.Background()
	request := &v1.PolicyFromSearchRequest{
		SearchParams: "Environment Key:z",
	}
	response, err := s.tested.PolicyFromSearch(ctx, request)
	s.NoError(err)
	s.False(response.GetHasNestedFields())
	s.Empty(response.GetAlteredSearchTerms())
	s.ElementsMatch(expectedPolicyGroup, response.GetPolicy().GetPolicySections()[0].GetPolicyGroups())
	expectedLifecycleStages := []storage.LifecycleStage{storage.LifecycleStage_DEPLOY}
	s.ElementsMatch(expectedLifecycleStages, response.GetPolicy().GetLifecycleStages())
}

// This test is the expected behavior after the sample mitre data injection is removed.
func (s *PolicyServiceTestSuite) TestMitreVectors() {
	s.policies.EXPECT().GetPolicy(gomock.Any(), "policy1").Return(&storage.Policy{
		Id: "policy1",
		MitreAttackVectors: []*storage.Policy_MitreAttackVectors{
			{
				Tactic:     "tactic1",
				Techniques: []string{"tech1"},
			},
			{
				Tactic:     "tactic2",
				Techniques: []string{"tech2"},
			},
		},
	}, true, nil)

	s.mitreVectorStore.EXPECT().Get("tactic1").Return(
		getFakeVector("tactic1", "tech1", "tech2", "tech3"), nil,
	)
	s.mitreVectorStore.EXPECT().Get("tactic2").Return(
		getFakeVector("tactic2", "tech1", "tech2"), nil,
	)

	response, err := s.tested.GetPolicyMitreVectors(context.Background(), &v1.GetPolicyMitreVectorsRequest{
		Id: "policy1",
	})
	s.NoError(err)
	s.ElementsMatch([]*storage.MitreAttackVector{
		getFakeVector("tactic1", "tech1"),
		getFakeVector("tactic2", "tech2"),
	}, response.GetVectors())
}

func getFakeVector(tactic string, techniques ...string) *storage.MitreAttackVector {
	resp := &storage.MitreAttackVector{
		Tactic: &storage.MitreTactic{
			Id:          tactic,
			Name:        tactic,
			Description: tactic,
		},
	}

	for _, technique := range techniques {
		resp.Techniques = append(resp.Techniques, &storage.MitreTechnique{
			Id:          technique,
			Name:        technique,
			Description: technique,
		})
	}

	return resp
}
