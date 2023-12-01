package compliancemanager

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	sensorMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	mockScanName = "mockScan"
)

type pipelineTestCase struct {
	desc                         string
	setMocksAndGetComplianceInfo func()
	complianceInfoGen            func() *storage.ComplianceIntegration
	isErrorTest                  bool
}

type processScanConfigTestCase struct {
	desc        string
	setMocks    func()
	isErrorTest bool
	expectedErr error
}

func TestComplianceManager(t *testing.T) {
	suite.Run(t, new(complianceManagerTestSuite))
}

type complianceManagerTestSuite struct {
	suite.Suite

	hasWriteCtx context.Context
	noAccessCtx context.Context

	mockCtrl      *gomock.Controller
	integrationDS *mocks.MockDataStore
	scanConfigDS  *scanConfigMocks.MockDataStore
	connectionMgr *sensorMocks.MockManager
	manager       Manager
}

func (suite *complianceManagerTestSuite) SetupSuite() {
	suite.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		suite.T().Skip("Skip tests when ComplianceEnhancements disabled")
		suite.T().SkipNow()
	}
}

func (suite *complianceManagerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
	suite.noAccessCtx = sac.WithNoAccess(context.Background())

	suite.integrationDS = mocks.NewMockDataStore(suite.mockCtrl)
	suite.scanConfigDS = scanConfigMocks.NewMockDataStore(suite.mockCtrl)
	suite.connectionMgr = sensorMocks.NewMockManager(suite.mockCtrl)
	suite.manager = New(suite.connectionMgr, suite.integrationDS, suite.scanConfigDS)
}

func (suite *complianceManagerTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *complianceManagerTestSuite) TestProcessComplianceOperatorInfo() {
	cases := []pipelineTestCase{
		{
			desc: "Error retrieving data",
			setMocksAndGetComplianceInfo: func() {
				query := search.NewQueryBuilder().
					AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()

				suite.integrationDS.EXPECT().GetComplianceIntegrations(gomock.Any(), query).Return(nil, errors.New("Unable to retrieve data")).Times(1)
			},
			complianceInfoGen: func() *storage.ComplianceIntegration {
				return &storage.ComplianceIntegration{
					Version:   "22",
					ClusterId: fixtureconsts.Cluster1,
					Namespace: fixtureconsts.Namespace1,
				}
			},
			isErrorTest: true,
		},
		{
			desc: "Add integration",
			setMocksAndGetComplianceInfo: func() {
				query := search.NewQueryBuilder().
					AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()

				suite.integrationDS.EXPECT().GetComplianceIntegrations(gomock.Any(), query).Return(nil, nil).Times(1)

				expectedInfo := &storage.ComplianceIntegration{
					Version:   "22",
					ClusterId: fixtureconsts.Cluster1,
					Namespace: fixtureconsts.Namespace1,
				}
				suite.integrationDS.EXPECT().AddComplianceIntegration(gomock.Any(), expectedInfo).Return(uuid.NewV4().String(), nil).Times(1)
			},
			complianceInfoGen: func() *storage.ComplianceIntegration {
				return &storage.ComplianceIntegration{
					Version:   "22",
					ClusterId: fixtureconsts.Cluster1,
					Namespace: fixtureconsts.Namespace1,
				}
			},
			isErrorTest: false,
		},
		{
			desc: "Update integration",
			setMocksAndGetComplianceInfo: func() {
				query := search.NewQueryBuilder().
					AddExactMatches(search.ClusterID, fixtureconsts.Cluster1).ProtoQuery()

				expectedInfo := &storage.ComplianceIntegration{
					Id:        uuid.NewV4().String(),
					Version:   "22",
					ClusterId: fixtureconsts.Cluster1,
					Namespace: fixtureconsts.Namespace1,
				}

				suite.integrationDS.EXPECT().GetComplianceIntegrations(gomock.Any(), query).Return([]*storage.ComplianceIntegration{expectedInfo}, nil).Times(1)

				suite.integrationDS.EXPECT().UpdateComplianceIntegration(gomock.Any(), expectedInfo).Return(nil).Times(1)
			},
			complianceInfoGen: func() *storage.ComplianceIntegration {
				return &storage.ComplianceIntegration{
					Version:   "22",
					ClusterId: fixtureconsts.Cluster1,
					Namespace: fixtureconsts.Namespace1,
				}
			},
			isErrorTest: false,
		},
	}

	for _, tc := range cases {
		suite.T().Run(tc.desc, func(t *testing.T) {
			// Setup the mock calls for this case
			tc.setMocksAndGetComplianceInfo()

			err := suite.manager.ProcessComplianceOperatorInfo(suite.hasWriteCtx, tc.complianceInfoGen())
			if tc.isErrorTest {
				suite.Require().NotNil(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *complianceManagerTestSuite) TestProcessScanRequest() {
	cases := []processScanConfigTestCase{
		{
			desc: "Successful creation of scan configuration",
			setMocks: func() {
				suite.scanConfigDS.EXPECT().ScanConfigurationExists(gomock.Any(), mockScanName).Return(false, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpsertScanConfiguration(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.connectionMgr.EXPECT().SendMessage(fixtureconsts.Cluster1, gomock.Any()).Return(nil).Times(1)
				suite.scanConfigDS.EXPECT().UpdateClusterStatus(gomock.Any(), gomock.Any(), fixtureconsts.Cluster1, "")
			},
			isErrorTest: false,
		},
		{
			desc: "Scan configuration already exists",
			setMocks: func() {
				suite.scanConfigDS.EXPECT().ScanConfigurationExists(gomock.Any(), mockScanName).Return(true, nil).Times(1)
			},
			isErrorTest: true,
			expectedErr: errors.Errorf("Scan Configuration named %q already exists.", mockScanName),
		},
		{
			desc: "Unable to store scan configuration",
			setMocks: func() {
				suite.scanConfigDS.EXPECT().ScanConfigurationExists(gomock.Any(), mockScanName).Return(false, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpsertScanConfiguration(gomock.Any(), gomock.Any()).Return(errors.Errorf("Unable to save scan config named %q", mockScanName)).Times(1)
			},
			isErrorTest: true,
			expectedErr: errors.Errorf("Unable to save scan config named %q", mockScanName),
		},
		{
			desc: "Error from sensor",
			setMocks: func() {
				suite.scanConfigDS.EXPECT().ScanConfigurationExists(gomock.Any(), mockScanName).Return(false, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpsertScanConfiguration(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.connectionMgr.EXPECT().SendMessage(fixtureconsts.Cluster1, gomock.Any()).Return(errors.New("Unable to process sensor message")).Times(1)
				suite.scanConfigDS.EXPECT().UpdateClusterStatus(gomock.Any(), gomock.Any(), fixtureconsts.Cluster1, "Unable to process sensor message")
			},
			isErrorTest: false,
		},
	}
	for _, tc := range cases {
		suite.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			config, err := suite.manager.ProcessScanRequest(suite.hasWriteCtx, getTestRec(), []string{fixtureconsts.Cluster1})
			if tc.isErrorTest {
				suite.Require().NotNil(err)
				suite.Require().Nil(config)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(config)
			}
		})
	}
}

func getTestRec() *storage.ComplianceOperatorScanConfigurationV2 {
	return &storage.ComplianceOperatorScanConfigurationV2{
		ScanName:               mockScanName,
		AutoApplyRemediations:  false,
		AutoUpdateRemediations: false,
		OneTimeScan:            false,
		Profiles: []*storage.ProfileShim{
			{
				ProfileId:   uuid.NewV4().String(),
				ProfileName: "ocp4-cis",
			},
		},
		StrictNodeScan: false,
	}
}
