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
	rr := httptest.NewRecorder()

	// configure storage mocks
	clusterIDs := []string{"testcluster"}
	standardIDs := []string{"CIS_Docker_v1_2_0"}
	csPair := compliance.ClusterStandardPair{
		ClusterID:  "testcluster",
		StandardID: "CIS_Docker_v1_2_0",
	}
	latestRunResultBatch := map[compliance.ClusterStandardPair]types.ResultsWithStatus{
		csPair: {
			LastSuccessfulResults: &storage.ComplianceRunResults{
				DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
					"deployment1": {},
				},
			},
		},
	}

	s.mockDS.EXPECT().GetLatestRunResultsBatch(s.hasReadCtx, clusterIDs, standardIDs, types.WithMessageStrings).Return(latestRunResultBatch, nil)

	handler := NewComplianceHandler(s.mockDS)
	handler.ServeHTTP(rr, req)

	s.Equal(rr.Body.String(), "")

}
