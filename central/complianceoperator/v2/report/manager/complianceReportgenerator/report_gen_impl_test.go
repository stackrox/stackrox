package complianceReportgenerator

import (
	"context"
	"testing"

	checkResults "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type ComplainceReportingTestSuite struct {
	suite.Suite
	mockCtrl  *gomock.Controller
	ctx       context.Context
	reportGen *complianceReportGeneratorImpl
}

func (s *ComplainceReportingTestSuite) SetupSuite() {
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	s.mockCtrl = gomock.NewController(s.T())

	s.reportGen = &complianceReportGeneratorImpl{
		checkResultsDS: checkResults.NewMockDataStore(s.mockCtrl),
	}
}

func TestComplianceReporting(t *testing.T) {
	suite.Run(t, new(ComplainceReportingTestSuite))
}

func (s *ComplainceReportingTestSuite) TestFormatReport() {

	_, err := s.reportGen.Format(s.getReportData())
	s.Require().NoError(err)

}

func (s *ComplainceReportingTestSuite) getReportData() map[string][]*resultRow {
	results := make(map[string][]*resultRow)
	results["cluster1"] = []*resultRow{{
		ClusterName: "test_cluster1",
		CheckName:   "test_check1",
		Profile:     "test_profile1",
		ControlRef:  "test_control_ref1",
		Description: "description1",
		Status:      "Pass",
		Remediation: "remediation1",
	},
		{
			ClusterName: "test_cluster2",
			CheckName:   "test_check2",
			Profile:     "test_profile2",
			ControlRef:  "test_control_ref2",
			Description: "description2",
			Status:      "Fail",
			Remediation: "remediation2",
		},
	}
	return results
}
