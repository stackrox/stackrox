package splunk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/mocks"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// This file contains tests for the /compliance endpoint

var (
	clusterID  = "compliance-test-id"
	clusterIDs = []string{clusterID}

	csPair = compliance.ClusterStandardPair{
		ClusterID:  clusterID,
		StandardID: "CIS_Kubernetes_v1_5",
	}
	latestRunResultBatch = map[compliance.ClusterStandardPair]types.ResultsWithStatus{
		csPair: {
			LastSuccessfulResults: storage.ComplianceRunResults_builder{
				RunMetadata: storage.ComplianceRunMetadata_builder{
					RunId:      "compliance-run-metadata-id",
					StandardId: "CIS_Kubernetes_v1_5",
					ClusterId:  clusterID,
				}.Build(),
				Domain: storage.ComplianceDomain_builder{
					Id: "compliance-test-id",
					Cluster: storage.ComplianceDomain_Cluster_builder{
						Name: clusterID,
					}.Build(),
					Deployments: map[string]*storage.ComplianceDomain_Deployment{
						"deployment1": storage.ComplianceDomain_Deployment_builder{
							Id:        "deployment1",
							Name:      "deployment1",
							Namespace: "dep-ns1",
						}.Build(),
					},
					Nodes: map[string]*storage.ComplianceDomain_Node{
						"node1": storage.ComplianceDomain_Node_builder{
							Id:   "node1",
							Name: "node1",
						}.Build(),
					},
				}.Build(),
				ClusterResults: storage.ComplianceRunResults_EntityResults_builder{
					ControlResults: map[string]*storage.ComplianceResultValue{
						"HIPAA_164:310_a_1": storage.ComplianceResultValue_builder{
							Evidence: []*storage.ComplianceResultValue_Evidence{storage.ComplianceResultValue_Evidence_builder{
								State:   storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
								Message: "Cluster has an image scanner in use",
							}.Build()},
							OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
						}.Build(),
					},
				}.Build(),
				DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
					"deployment1": storage.ComplianceRunResults_EntityResults_builder{
						ControlResults: map[string]*storage.ComplianceResultValue{
							"CIS_Kubernetes_v1_5:5_6": storage.ComplianceResultValue_builder{
								Evidence: []*storage.ComplianceResultValue_Evidence{storage.ComplianceResultValue_Evidence_builder{
									State:   storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
									Message: "Container has no ssh process running",
								}.Build()},
								OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
							}.Build(),
						},
					}.Build(),
				},
				NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
					"node1": storage.ComplianceRunResults_EntityResults_builder{
						ControlResults: map[string]*storage.ComplianceResultValue{
							"CIS_Kubernetes_v1_5:1_1_2": storage.ComplianceResultValue_builder{
								Evidence: []*storage.ComplianceResultValue_Evidence{storage.ComplianceResultValue_Evidence_builder{
									State:   storage.ComplianceState_COMPLIANCE_STATE_SKIP,
									Message: "Node does not use Docker container runtime",
								}.Build()},
								OverallState: storage.ComplianceState_COMPLIANCE_STATE_SKIP,
							}.Build(),
						},
					}.Build(),
				},
				MachineConfigResults: map[string]*storage.ComplianceRunResults_EntityResults{
					"ocp4-cis-node-master": storage.ComplianceRunResults_EntityResults_builder{
						ControlResults: map[string]*storage.ComplianceResultValue{
							"ocp4-cis-node:file-owner-worker-kubeconfig": storage.ComplianceResultValue_builder{
								Evidence: []*storage.ComplianceResultValue_Evidence{storage.ComplianceResultValue_Evidence_builder{
									State:   storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
									Message: "Pass for ocp4-cis-node-master-file-owner-worker-kubeconfig.",
								}.Build()},
								OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		},
	}
)

func TestSplunkComplianceAPI(t *testing.T) {
	suite.Run(t, &splunkComplianceAPITestSuite{})
}

type splunkComplianceAPITestSuite struct {
	suite.Suite

	hasReadCtx context.Context

	mockCtrl *gomock.Controller
	mockDS   *mocks.MockDataStore
}

func (s *splunkComplianceAPITestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.mockCtrl = gomock.NewController(s.T())
	s.mockDS = mocks.NewMockDataStore(s.mockCtrl)
}

func (s *splunkComplianceAPITestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *splunkComplianceAPITestSuite) TestComplianceAPIResults() {
	// set up http mocks
	req, err := http.NewRequest("GET", "/api/splunk/ta/compliance", nil)
	require.NoError(s.T(), err)
	responseRecorder := httptest.NewRecorder()

	// set up override for getClusterIDs
	getMockClusterIDs := func(ctx context.Context) ([]string, error) {
		return clusterIDs, nil
	}

	// configure storage mock
	s.mockDS.EXPECT().GetLatestRunResultsBatch(req.Context(), clusterIDs, gomock.Any(), types.RequireMessageStrings).Return(latestRunResultBatch, nil).AnyTimes()

	// use internal function that accepts an additional parameter to simplify mocking/testing
	handler := newComplianceHandler(s.mockDS, getMockClusterIDs)

	handler.ServeHTTP(responseRecorder, req)
	responseBody := responseRecorder.Body.String()

	// Primarily, we want to ensure that all RunResults are handed to the SplunkAPI results.
	// By testing the return format, we additionally ensure that the returned data is complete and well formatted.
	expectedCompliance := []struct {
		name           string
		expectedResult string
	}{
		{
			name:           "Cluster Results",
			expectedResult: "{\"standard\":\"CIS Kubernetes v1.5\",\"cluster\":\"compliance-test-id\",\"namespace\":\"\",\"objectType\":\"Cluster\",\"objectName\":\"compliance-test-id\",\"control\":\"HIPAA_164:310_a_1\",\"state\":\"Pass\",\"evidence\":\"(Pass) Cluster has an image scanner in use\"}",
		},
		{
			name:           "Deployment Results",
			expectedResult: "{\"standard\":\"CIS Kubernetes v1.5\",\"cluster\":\"compliance-test-id\",\"namespace\":\"dep-ns1\",\"objectType\":\"Deployment\",\"objectName\":\"deployment1\",\"control\":\"CIS_Kubernetes_v1_5:5_6\",\"state\":\"Pass\",\"evidence\":\"(Pass) Container has no ssh process running\"}",
		},
		{
			name:           "Node Results",
			expectedResult: "{\"standard\":\"CIS Kubernetes v1.5\",\"cluster\":\"compliance-test-id\",\"namespace\":\"\",\"objectType\":\"Node\",\"objectName\":\"node1\",\"control\":\"1.1.2\",\"state\":\"N/A\",\"evidence\":\"(N/A) Node does not use Docker container runtime\"}",
		},
		{
			name:           "Machine Config Results",
			expectedResult: "{\"standard\":\"CIS Kubernetes v1.5\",\"cluster\":\"compliance-test-id\",\"namespace\":\"\",\"objectType\":\"Machine Config\",\"objectName\":\"ocp4-cis-node-master\",\"control\":\"ocp4-cis-node:file-owner-worker-kubeconfig\",\"state\":\"Pass\",\"evidence\":\"(Pass) Pass for ocp4-cis-node-master-file-owner-worker-kubeconfig.\"}",
		},
	}

	for _, result := range expectedCompliance {
		s.Run(result.name, func() {
			s.Contains(responseBody, result.expectedResult, "Response did not contain expected results")
		})
	}
}
