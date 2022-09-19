package splunk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance"
	"github.com/stackrox/rox/central/compliance/datastore/mocks"
	"github.com/stackrox/rox/central/compliance/datastore/types"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// This file contains tests for the /compliance endpoint

func TestSplunkComplianceAPI(t *testing.T) {
	//t.Parallel()
	suite.Run(t, &splunkComplianceAPITestSuite{})
}

type splunkComplianceAPITestSuite struct {
	suite.Suite

	hasReadCtx  context.Context
	hasWriteCtx context.Context

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

func (s *splunkComplianceAPITestSuite) TestCISDockerResults() {
	// set up http mocks
	req, err := http.NewRequest("GET", "/api/splunk/ta/compliance", nil)
	require.NoError(s.T(), err)
	responseRecorder := httptest.NewRecorder()

	// configure storage mocks
	clusterIDs := []string{"compliance-test-id"}
	//standardIDs := []string{"CIS_Docker_v1_2_0"}
	csPair := compliance.ClusterStandardPair{
		ClusterID:  "compliance-test-id",
		StandardID: "CIS_Docker_v1_2_0",
	}
	latestRunResultBatch := map[compliance.ClusterStandardPair]types.ResultsWithStatus{
		csPair: {
			LastSuccessfulResults: &storage.ComplianceRunResults{
				// TODO: Add additional fields
				Domain: &storage.ComplianceDomain{
					Id: "compliance-test-id",
					Cluster: &storage.Cluster{
						Name: "compliance_test cluster",
					},
				},
				DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
					"deployment1": {},
				},
				MachineConfigResults: map[string]*storage.ComplianceRunResults_EntityResults{
					"ocp4-cis-node-master": {
						ControlResults: map[string]*storage.ComplianceResultValue{
							"ocp4-cis-node:file-owner-worker-kubeconfig": {
								Evidence: []*storage.ComplianceResultValue_Evidence{{
									State:   storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
									Message: "Pass for ocp4-cis-node-master-file-owner-worker-kubeconfig.",
								}},
								OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
							},
						},
					},
				},
			},
		},
	}

	s.mockDS.EXPECT().GetLatestRunResultsBatch(req.Context(), clusterIDs, gomock.Any(), types.RequireMessageStrings).Return(latestRunResultBatch, nil).AnyTimes()

	handler := NewComplianceHandler(s.mockDS)

	getClusterIDs = func(ctx context.Context) ([]string, error) {
		return clusterIDs, nil
	}
	handler.ServeHTTP(responseRecorder, req)

	s.Equal("", responseRecorder.Body.String())

}
