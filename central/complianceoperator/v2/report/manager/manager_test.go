package manager

import (
	"context"
	"testing"

	profileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	reportGen "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/complianceReportgenerator/mocks"
	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	scanMocks "github.com/stackrox/rox/central/complianceoperator/v2/scans/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type ManagerTestSuite struct {
	suite.Suite
	mockCtrl         *gomock.Controller
	ctx              context.Context
	datastore        *scanConfigurationDS.MockDataStore
	scanDataStore    *scanMocks.MockDataStore
	profileDataStore *profileMocks.MockDataStore
	reportGen        *reportGen.MockComplianceReportGenerator
}

func (m *ManagerTestSuite) SetupSuite() {
	m.T().Setenv(features.ComplianceReporting.EnvVar(), "true")
	m.ctx = sac.WithAllAccess(context.Background())
}

func (m *ManagerTestSuite) SetupTest() {
	m.mockCtrl = gomock.NewController(m.T())
	m.datastore = scanConfigurationDS.NewMockDataStore(m.mockCtrl)
	m.scanDataStore = scanMocks.NewMockDataStore(m.mockCtrl)
	m.profileDataStore = profileMocks.NewMockDataStore(m.mockCtrl)
	m.reportGen = reportGen.NewMockComplianceReportGenerator(m.mockCtrl)
}

func TestComplianceReportManager(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

func (m *ManagerTestSuite) TestSubmitReportRequest() {
	manager := New(m.datastore, m.scanDataStore, m.profileDataStore, m.reportGen)
	reportRequest := &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: "test_scan_config",
		Id:             "test_scan_config",
	}
	err := manager.SubmitReportRequest(m.ctx, reportRequest)
	m.Require().NoError(err)
	err = manager.SubmitReportRequest(m.ctx, reportRequest)
	m.Require().Error(err)
}

func (m *ManagerTestSuite) TearDownTest() {
	m.mockCtrl.Finish()
}
