package manager

import (
	"context"
	"testing"

	scanConfigurationDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type ManagerTestSuite struct {
	suite.Suite
	mockCtrl  *gomock.Controller
	ctx       context.Context
	datastore *scanConfigurationDS.MockDataStore
}

func (m *ManagerTestSuite) SetupSuite() {
	if features.ComplianceReporting.Enabled() {
		return
	}
	m.ctx = sac.WithAllAccess(context.Background())

}

func (m *ManagerTestSuite) SetupTest() {
	m.mockCtrl = gomock.NewController(m.T())
	m.datastore = scanConfigurationDS.NewMockDataStore(m.mockCtrl)

}

func TestComplianceProfileService(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

func (m *ManagerTestSuite) TestSubmitReportRequest() {
	manager := New(m.datastore)
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
