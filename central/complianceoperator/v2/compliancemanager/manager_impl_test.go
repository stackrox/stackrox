package compliancemanager

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	clusterDatastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	resultsMocks "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/integration/datastore/mocks"
	profileMocks "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	"github.com/stackrox/rox/central/convert/internaltov2storage"
	sensorMocks "github.com/stackrox/rox/central/sensor/service/connection/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	mockScanName = "mockScan"
	mockScanID   = "mockScanID"
)

var expectedComplianceOperatorScanConfigurationV2 = &storage.ComplianceOperatorScanConfigurationV2{
	ScanConfigName: mockScanName,
	Profiles:       []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{{ProfileName: "ocp4-cis"}},
}

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
	expectedErr string
	testContext context.Context
	clusters    []string
	testRequest *storage.ComplianceOperatorScanConfigurationV2
}

func TestComplianceManager(t *testing.T) {
	suite.Run(t, new(complianceManagerTestSuite))
}

type complianceManagerTestSuite struct {
	suite.Suite

	hasWriteCtx  context.Context
	noAccessCtx  context.Context
	testContexts map[string]context.Context

	mockCtrl         *gomock.Controller
	integrationDS    *mocks.MockDataStore
	scanConfigDS     *scanConfigMocks.MockDataStore
	profileDS        *profileMocks.MockDataStore
	resultsDS        *resultsMocks.MockDataStore
	connectionMgr    *sensorMocks.MockManager
	clusterDatastore *clusterDatastoreMocks.MockDataStore
	manager          Manager
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
			sac.ResourceScopeKeys(resources.Compliance)))
	suite.noAccessCtx = sac.WithNoAccess(context.Background())
	suite.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), suite.T(), resources.Compliance)

	suite.integrationDS = mocks.NewMockDataStore(suite.mockCtrl)
	suite.scanConfigDS = scanConfigMocks.NewMockDataStore(suite.mockCtrl)
	suite.connectionMgr = sensorMocks.NewMockManager(suite.mockCtrl)
	suite.clusterDatastore = clusterDatastoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.profileDS = profileMocks.NewMockDataStore(suite.mockCtrl)
	suite.resultsDS = resultsMocks.NewMockDataStore(suite.mockCtrl)
	suite.manager = New(suite.connectionMgr, suite.integrationDS, suite.scanConfigDS, suite.clusterDatastore, suite.profileDS, suite.resultsDS)
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
					Version:             "22",
					ClusterId:           fixtureconsts.Cluster1,
					ComplianceNamespace: fixtureconsts.Namespace1,
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
					Version:             "22",
					ClusterId:           fixtureconsts.Cluster1,
					ComplianceNamespace: fixtureconsts.Namespace1,
				}
				suite.integrationDS.EXPECT().AddComplianceIntegration(gomock.Any(), expectedInfo).Return(uuid.NewV4().String(), nil).Times(1)
			},
			complianceInfoGen: func() *storage.ComplianceIntegration {
				return &storage.ComplianceIntegration{
					Version:             "22",
					ClusterId:           fixtureconsts.Cluster1,
					ComplianceNamespace: fixtureconsts.Namespace1,
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
					Id:                  uuid.NewV4().String(),
					Version:             "22",
					ClusterId:           fixtureconsts.Cluster1,
					ComplianceNamespace: fixtureconsts.Namespace1,
				}

				suite.integrationDS.EXPECT().GetComplianceIntegrations(gomock.Any(), query).Return([]*storage.ComplianceIntegration{expectedInfo}, nil).Times(1)

				suite.integrationDS.EXPECT().UpdateComplianceIntegration(gomock.Any(), expectedInfo).Return(nil).Times(1)
			},
			complianceInfoGen: func() *storage.ComplianceIntegration {
				return &storage.ComplianceIntegration{
					Version:             "22",
					ClusterId:           fixtureconsts.Cluster1,
					ComplianceNamespace: fixtureconsts.Namespace1,
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
			desc:        "Successful creation of scan configuration",
			testRequest: getTestRecNoID(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.profileDS.EXPECT().SearchProfiles(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return([]*storage.ComplianceOperatorProfileV2{
					getTestProfile("ocp4-cis", "1.0.0", "platform", "ocp4", testconsts.Cluster1, 1),
				}, nil).Times(1)
				suite.scanConfigDS.EXPECT().GetScanConfigurationByName(suite.testContexts[testutils.UnrestrictedReadWriteCtx], mockScanName).Return(nil, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpsertScanConfiguration(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return(nil).Times(1)
				suite.connectionMgr.EXPECT().SendMessage(testconsts.Cluster1, gomock.Any()).Return(nil).Times(1)
				suite.clusterDatastore.EXPECT().GetClusterName(gomock.Any(), gomock.Any()).Return("test_cluster", true, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpdateClusterStatus(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), testconsts.Cluster1, "", "test_cluster")
			},
			isErrorTest: false,
		},
		{
			desc:        "Successful creation of scan configuration with valid profiles",
			testRequest: getTestRecNoIDValidProfile(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().GetScanConfigurationByName(gomock.Any(), mockScanName).Return(nil, nil).Times(1)
				suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.profileDS.EXPECT().SearchProfiles(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return([]*storage.ComplianceOperatorProfileV2{
					getTestProfile("ocp4-cis", "1.0.0", "platform", "ocp4", testconsts.Cluster1, 1),
					getTestProfile("rhcos4-cis", "1.0.0", "node", "rhcos4", testconsts.Cluster1, 1),
				}, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpsertScanConfiguration(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return(nil).Times(1)
				suite.connectionMgr.EXPECT().SendMessage(testconsts.Cluster1, gomock.Any()).Return(nil).Times(1)
				suite.clusterDatastore.EXPECT().GetClusterName(gomock.Any(), gomock.Any()).Return("test_cluster", true, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpdateClusterStatus(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), testconsts.Cluster1, "", "test_cluster")
			},
			isErrorTest: false,
		},
		{
			desc:        "Invalid profiles in scan configuration",
			testRequest: getTestRecNoIDInvalidProfile(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().GetScanConfigurationByName(gomock.Any(), mockScanName).Return(nil, nil).Times(1)
				suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.profileDS.EXPECT().SearchProfiles(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return([]*storage.ComplianceOperatorProfileV2{
					getTestProfile("ocp4-cis-node", "1.0.0", "node", "ocp4", testconsts.Cluster1, 1),
					getTestProfile("rhcos4-cis", "1.0.0", "node", "rhcos4", testconsts.Cluster1, 1),
				}, nil).Times(1)
			},
			isErrorTest: true,
			expectedErr: fmt.Sprintf("Unable to create scan configuration named %q.", mockScanName),
		},
		{
			desc:        "Scan configuration already exists",
			testRequest: getTestRecNoID(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().GetScanConfigurationByName(gomock.Any(), mockScanName).Return(getTestRec(), nil).Times(1)
			},
			isErrorTest: true,
			expectedErr: fmt.Sprintf("Scan configuration named %q already exists.", mockScanName),
		},
		{
			desc:        "Scan configuration has duplicate profiles",
			testRequest: getTestRecNoID(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().GetScanConfigurationByName(suite.testContexts[testutils.UnrestrictedReadWriteCtx], mockScanName).Return(nil, nil).Times(1)
				suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.Errorf("Duplicated profiles found in current or existing scan configurations: %q.", mockScanName)).Times(1)
			},
			isErrorTest: true,
			expectedErr: fmt.Sprintf("Duplicated profiles found in current or existing scan configurations: %q.", mockScanName),
		},
		{
			desc:        "Unable to store scan configuration",
			testRequest: getTestRecNoID(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.profileDS.EXPECT().SearchProfiles(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return([]*storage.ComplianceOperatorProfileV2{
					getTestProfile("ocp4-cis", "1.0.0", "platform", "ocp4", testconsts.Cluster1, 1),
				}, nil).Times(1)
				suite.scanConfigDS.EXPECT().GetScanConfigurationByName(suite.testContexts[testutils.UnrestrictedReadWriteCtx], mockScanName).Return(nil, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpsertScanConfiguration(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return(errors.Errorf("Unable to save scan config named %q", mockScanName)).Times(1)
			},
			isErrorTest: true,
			expectedErr: fmt.Sprintf("Unable to save scan configuration named %q", mockScanName),
		},
		{
			desc:        "Error from sensor",
			testRequest: getTestRecNoID(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.scanConfigDS.EXPECT().GetScanConfigurationByName(suite.testContexts[testutils.UnrestrictedReadWriteCtx], mockScanName).Return(nil, nil).Times(1)
				suite.profileDS.EXPECT().SearchProfiles(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return([]*storage.ComplianceOperatorProfileV2{
					getTestProfile("ocp4-cis", "1.0.0", "platform", "ocp4", testconsts.Cluster1, 1),
				}, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpsertScanConfiguration(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return(nil).Times(1)
				suite.connectionMgr.EXPECT().SendMessage(testconsts.Cluster1, gomock.Any()).Return(errors.New("Unable to process sensor message")).Times(1)
				suite.clusterDatastore.EXPECT().GetClusterName(gomock.Any(), gomock.Any()).Return("test_cluster", true, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpdateClusterStatus(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), testconsts.Cluster1, "Unable to process sensor message", "test_cluster")
			},
			isErrorTest: false,
		},
		{
			desc:        "Error due to not having write access",
			testRequest: getTestRecNoID(),
			testContext: suite.testContexts[testutils.UnrestrictedReadCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
			},
			isErrorTest: true,
			expectedErr: "access to resource denied",
		},
		{
			desc:        "Error due to only having write access to one of the clusters",
			testRequest: getTestRecNoID(),
			testContext: suite.testContexts[testutils.Cluster1ReadWriteCtx],
			clusters:    []string{testconsts.Cluster1, testconsts.Cluster2},
			setMocks: func() {
			},
			isErrorTest: true,
			expectedErr: "access to resource denied",
		},
		{
			desc:        "Failure try to re-add a scan configuration",
			testRequest: getTestRec(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks:    func() {},
			isErrorTest: true,
			expectedErr: "The scan configuration already exists and cannot be added.  ID \"mockScanID\" and name \"mockScan\"",
		},
		{
			desc:        "Creating scan configuration with invalid cluster ID fails",
			testRequest: getTestRecNoID(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.scanConfigDS.EXPECT().GetScanConfigurationByName(suite.testContexts[testutils.UnrestrictedReadWriteCtx], mockScanName).Return(nil, nil).Times(1)
				suite.profileDS.EXPECT().SearchProfiles(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return([]*storage.ComplianceOperatorProfileV2{
					getTestProfile("ocp4-cis", "1.0.0", "platform", "ocp4", testconsts.Cluster1, 1),
				}, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpsertScanConfiguration(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return(nil).Times(1)
				suite.connectionMgr.EXPECT().SendMessage(testconsts.Cluster1, gomock.Any()).Return(errors.New("Unable to process sensor message")).Times(1)
				suite.clusterDatastore.EXPECT().GetClusterName(gomock.Any(), gomock.Any()).Return("", false, nil).Times(1)
			},
			isErrorTest: true,
			expectedErr: "Unable to save scan configuration status for scan named \"mockScan\": could not pull config for cluster \"aaaaaaaa-bbbb-4011-0000-111111111111\" because it does not exist",
		},
	}
	for _, tc := range cases {
		suite.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			config, err := suite.manager.ProcessScanRequest(tc.testContext, tc.testRequest, tc.clusters)
			if tc.isErrorTest {
				suite.Require().ErrorContains(err, tc.expectedErr)
				suite.Require().Nil(config)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(config)
			}
		})
	}
}

func (suite *complianceManagerTestSuite) TestUpdateScanRequest() {
	cases := []processScanConfigTestCase{
		{
			desc:        "Unable to update due to no scan config ID",
			testRequest: getTestRecNoID(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks:    func() {},
			isErrorTest: true,
			expectedErr: "Scan Configuration ID is required for an update",
		},
		{
			desc:        "Scan configuration has duplicate profiles",
			testRequest: getTestRec(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), mockScanID).Return(getTestRec(), true, nil).Times(1)
				suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.Errorf("Duplicated profiles found in current or existing scan configurations: %q.", mockScanName)).Times(1)
			},
			isErrorTest: true,
			expectedErr: fmt.Sprintf("Duplicated profiles found in current or existing scan configurations: %q.", mockScanName),
		},
		{
			desc:        "Unable to store scan configuration",
			testRequest: getTestRec(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), mockScanID).Return(getTestRec(), true, nil).Times(1)
				suite.profileDS.EXPECT().SearchProfiles(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return([]*storage.ComplianceOperatorProfileV2{
					getTestProfile("ocp4-cis", "1.0.0", "platform", "ocp4", testconsts.Cluster1, 1),
				}, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpsertScanConfiguration(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return(errors.Errorf("Unable to save scan config named %q", mockScanName)).Times(1)
			},
			isErrorTest: true,
			expectedErr: fmt.Sprintf("Unable to save scan configuration named %q", mockScanName),
		},
		{
			desc:        "Error from sensor",
			testRequest: getTestRec(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), mockScanID).Return(getTestRec(), true, nil).Times(1)
				suite.profileDS.EXPECT().SearchProfiles(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return([]*storage.ComplianceOperatorProfileV2{
					getTestProfile("ocp4-cis", "1.0.0", "platform", "ocp4", testconsts.Cluster1, 1),
				}, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpsertScanConfiguration(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return(nil).Times(1)
				suite.connectionMgr.EXPECT().SendMessage(testconsts.Cluster1, gomock.Any()).Return(errors.New("Unable to process sensor message")).Times(1)
				suite.clusterDatastore.EXPECT().GetClusterName(gomock.Any(), gomock.Any()).Return("test_cluster", true, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpdateClusterStatus(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), testconsts.Cluster1, "Unable to process sensor message", "test_cluster")
			},
			isErrorTest: false,
		},
		{
			desc:        "Error due to not having write access",
			testRequest: getTestRecNoID(),
			testContext: suite.testContexts[testutils.UnrestrictedReadCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
			},
			isErrorTest: true,
			expectedErr: fmt.Sprintf("Scan Configuration ID is required for an update, %+v", expectedComplianceOperatorScanConfigurationV2),
		},
		{
			desc:        "Error due to only having write access to one of the clusters",
			testRequest: getTestRecNoID(),
			testContext: suite.testContexts[testutils.Cluster1ReadWriteCtx],
			clusters:    []string{testconsts.Cluster1, testconsts.Cluster2},
			setMocks: func() {
			},
			isErrorTest: true,
			expectedErr: fmt.Sprintf("Scan Configuration ID is required for an update, %+v", expectedComplianceOperatorScanConfigurationV2),
		},
		{
			desc:        "Successful update of scan configuration",
			testRequest: getTestRec(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), mockScanID).Return(getTestRec(), true, nil).Times(1)
				suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.profileDS.EXPECT().SearchProfiles(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return([]*storage.ComplianceOperatorProfileV2{
					getTestProfile("ocp4-cis", "1.0.0", "platform", "ocp4", testconsts.Cluster1, 1),
				}, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpsertScanConfiguration(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return(nil).Times(1)
				suite.connectionMgr.EXPECT().SendMessage(testconsts.Cluster1, gomock.Any()).Return(nil).Times(1)
				suite.clusterDatastore.EXPECT().GetClusterName(gomock.Any(), gomock.Any()).Return("test_cluster", true, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpdateClusterStatus(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), testconsts.Cluster1, "", "test_cluster")
			},
			isErrorTest: false,
		},
		{
			desc:        "Successful update of scan configuration that removes cluster 2",
			testRequest: getTestRec(),
			testContext: suite.testContexts[testutils.UnrestrictedReadWriteCtx],
			clusters:    []string{testconsts.Cluster1},
			setMocks: func() {
				suite.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), mockScanID).Return(getTestRecMultiCluster(), true, nil).Times(1)
				suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.profileDS.EXPECT().SearchProfiles(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return([]*storage.ComplianceOperatorProfileV2{
					getTestProfile("ocp4-cis", "1.0.0", "platform", "ocp4", testconsts.Cluster1, 1),
				}, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpsertScanConfiguration(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any()).Return(nil).Times(1)
				suite.scanConfigDS.EXPECT().RemoveClusterStatus(gomock.Any(), mockScanID, testconsts.Cluster2).Return(nil).Times(1)
				suite.connectionMgr.EXPECT().SendMessage(testconsts.Cluster1, gomock.Any()).Return(nil).Times(1)
				suite.connectionMgr.EXPECT().SendMessage(testconsts.Cluster2, gomock.Any()).Return(nil).Times(1)
				suite.resultsDS.EXPECT().DeleteResultsByScanConfigAndCluster(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				suite.clusterDatastore.EXPECT().GetClusterName(gomock.Any(), gomock.Any()).Return("test_cluster", true, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpdateClusterStatus(suite.testContexts[testutils.UnrestrictedReadWriteCtx], gomock.Any(), testconsts.Cluster1, "", "test_cluster")
			},
			isErrorTest: false,
		},
	}
	for _, tc := range cases {
		suite.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			config, err := suite.manager.UpdateScanRequest(tc.testContext, tc.testRequest, tc.clusters)
			if tc.isErrorTest {
				suite.Require().ErrorContains(err, tc.expectedErr)
				suite.Require().Nil(config)
			} else {
				suite.Require().NoError(err)
				suite.Require().NotNil(config)
			}
		})
	}
}

func (suite *complianceManagerTestSuite) TestDeleteScanConfiguration() {
	cases := []processScanConfigTestCase{
		{
			desc: "Successful delection of scan configuration",
			setMocks: func() {
				suite.scanConfigDS.EXPECT().DeleteScanConfiguration(gomock.Any(), mockScanID).Return(mockScanName,
					nil).Times(1)
				suite.connectionMgr.EXPECT().BroadcastMessage(gomock.Any()).Times(1)
			},
			isErrorTest: false,
		},
		{
			desc: "Error from delection of scan configuration",
			setMocks: func() {
				suite.scanConfigDS.EXPECT().DeleteScanConfiguration(gomock.Any(), mockScanID).Return(mockScanName,
					errors.New("Unable to delete scan configuration")).Times(1)
			},
			isErrorTest: true,
			expectedErr: "Unable to delete scan configuration",
		},
		{
			desc: "Empty scan configuration name",
			setMocks: func() {
				suite.scanConfigDS.EXPECT().DeleteScanConfiguration(gomock.Any(), mockScanID).Return("",
					nil).Times(1)
			},
			isErrorTest: true,
			expectedErr: fmt.Sprintf("Unable to find scan configuration name for ID %q", mockScanID),
		},
	}
	for _, tc := range cases {
		suite.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			err := suite.manager.DeleteScan(suite.hasWriteCtx, getTestRec().Id)
			if tc.isErrorTest {
				suite.Require().NotNil(err)
				suite.Require().ErrorContains(err, tc.expectedErr)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func getTestProfile(profileName string, version string, platform string, product string, clusterID string, ruleCount int) *storage.ComplianceOperatorProfileV2 {
	var rules []*storage.ComplianceOperatorProfileV2_Rule

	if ruleCount > 0 {
		rules = make([]*storage.ComplianceOperatorProfileV2_Rule, 0, ruleCount)
		for i := 0; i < ruleCount; i++ {
			rules = append(rules, &storage.ComplianceOperatorProfileV2_Rule{
				RuleName: fmt.Sprintf("name-%d", i),
			})
		}
	}

	return &storage.ComplianceOperatorProfileV2{
		Id:             uuid.NewV4().String(),
		ProfileId:      uuid.NewV4().String(),
		Name:           profileName,
		ProfileVersion: version,
		ProductType:    platform,
		Standard:       profileName,
		Description:    "this is a test",
		Labels:         nil,
		Annotations:    nil,
		Product:        product,
		ClusterId:      clusterID,
		Title:          "A Title",
		Rules:          rules,
	}
}

func getTestRec() *storage.ComplianceOperatorScanConfigurationV2 {
	return getTestRecWithClustersAndProfiles(mockScanID, []string{testconsts.Cluster1}, []string{"ocp4-cis"})
}

func getTestRecMultiCluster() *storage.ComplianceOperatorScanConfigurationV2 {
	return getTestRecWithClustersAndProfiles(mockScanID, []string{testconsts.Cluster1, testconsts.Cluster2}, []string{"ocp4-cis"})
}

func (suite *complianceManagerTestSuite) TestProcessRescanRequest() {
	multiCluster := getTestRec()
	multiCluster.Clusters = append(multiCluster.Clusters, &storage.ComplianceOperatorScanConfigurationV2_Cluster{ClusterId: testconsts.Cluster3})
	cases := []processScanConfigTestCase{
		{
			desc: "Rerun existing scan config succeeds",
			setMocks: func() {
				suite.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), mockScanID).Return(getTestRec(), true, nil).Times(1)
				suite.connectionMgr.EXPECT().SendMessage(testconsts.Cluster1, gomock.Any()).Return(nil).Times(1)
			},
			isErrorTest: false,
		},
		{
			desc: "Rerun non-existent scan config fails",
			setMocks: func() {
				suite.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), mockScanID).Return(nil, false, nil).Times(1)
			},
			isErrorTest: true,
		},
		{
			desc: "Rerun scan config fails when data store returns an error finding scan config",
			setMocks: func() {
				suite.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), mockScanID).Return(nil, false, errors.New("Unable to retrieve data")).Times(1)
			},
			isErrorTest: true,
		},
		{
			desc: "Rerun scan config continues when sensor message fails and logs message",
			setMocks: func() {
				suite.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), mockScanID).Return(multiCluster, true, nil).Times(1)
				suite.connectionMgr.EXPECT().SendMessage(testconsts.Cluster1, gomock.Any()).Return(errors.New("Failed to send message to sensor")).Times(1)
				suite.clusterDatastore.EXPECT().GetClusterName(gomock.Any(), gomock.Any()).Return("test_cluster", true, nil).Times(1)
				suite.scanConfigDS.EXPECT().UpdateClusterStatus(gomock.Any(), mockScanID, testconsts.Cluster1, "Failed to send message to sensor", "test_cluster").Times(1)
				suite.connectionMgr.EXPECT().SendMessage(testconsts.Cluster3, gomock.Any()).Return(nil).Times(1)
			},
			isErrorTest: false,
		},
	}
	for _, tc := range cases {
		suite.T().Run(tc.desc, func(t *testing.T) {
			tc.setMocks()

			err := suite.manager.ProcessRescanRequest(suite.hasWriteCtx, mockScanID)
			if tc.isErrorTest {
				suite.Require().NotNil(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func getTestRecNoID() *storage.ComplianceOperatorScanConfigurationV2 {
	return getTestRecWithClustersAndProfiles("", []string{testconsts.Cluster1}, []string{"ocp4-cis"})
}

func getTestRecNoIDInvalidProfile() *storage.ComplianceOperatorScanConfigurationV2 {
	return getTestRecWithClustersAndProfiles("", []string{testconsts.Cluster1}, []string{"ocp4-cis-node", "rhcos4-cis"})
}

func getTestRecNoIDValidProfile() *storage.ComplianceOperatorScanConfigurationV2 {
	return getTestRecWithClustersAndProfiles("", []string{testconsts.Cluster1}, []string{"ocp4-cis", "rhcos4-cis"})
}

func getTestRecWithClustersAndProfiles(scanID string, clusterIDs []string, profileNames []string) *storage.ComplianceOperatorScanConfigurationV2 {
	clusters := make([]*storage.ComplianceOperatorScanConfigurationV2_Cluster, 0, len(clusterIDs))
	for _, clusterID := range clusterIDs {
		clusters = append(clusters, &storage.ComplianceOperatorScanConfigurationV2_Cluster{
			ClusterId: clusterID,
		})
	}

	profiles := make([]*storage.ComplianceOperatorScanConfigurationV2_ProfileName, 0, len(profileNames))
	for _, profileName := range profileNames {
		profiles = append(profiles, &storage.ComplianceOperatorScanConfigurationV2_ProfileName{
			ProfileName: profileName,
		})
	}

	return &storage.ComplianceOperatorScanConfigurationV2{
		Id:                     scanID,
		ScanConfigName:         mockScanName,
		AutoApplyRemediations:  false,
		AutoUpdateRemediations: false,
		OneTimeScan:            false,
		Profiles:               profiles,
		Clusters:               clusters,
		StrictNodeScan:         false,
	}
}

func (suite *complianceManagerTestSuite) TestRemoveObsoleteResultsByClusters() {
	cases := map[string]struct {
		oldClusterIDs []string
		newClusterIDs []string
		profileNames  []string
	}{
		"no changes": {
			oldClusterIDs: []string{testconsts.Cluster1},
			newClusterIDs: []string{testconsts.Cluster1},
			profileNames:  []string{"ocp4-cis"},
		},
		"new cluster": {
			oldClusterIDs: []string{testconsts.Cluster1},
			newClusterIDs: []string{testconsts.Cluster1, testconsts.Cluster2},
			profileNames:  []string{"ocp4-cis"},
		},
		"remove 2 clusters": {
			oldClusterIDs: []string{testconsts.Cluster1, testconsts.Cluster2, testconsts.Cluster3},
			newClusterIDs: []string{testconsts.Cluster2},
			profileNames:  []string{"ocp4-cis", "ocp4-cis-node"},
		},
	}

	for testName, tc := range cases {
		suite.T().Run(testName, func(tt *testing.T) {
			oldScanConfig := getTestRecWithClustersAndProfiles(mockScanID, tc.oldClusterIDs, tc.profileNames)
			newScanConfig := getTestRecWithClustersAndProfiles(mockScanID, tc.newClusterIDs, tc.profileNames)
			expectedRemovedClusters := set.NewStringSet(tc.oldClusterIDs...).Difference(set.NewStringSet(tc.newClusterIDs...)).AsSlice()

			// Default scan update calls.
			suite.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), mockScanID).Return(oldScanConfig, true, nil).Times(1)
			suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
			testProfiles := make([]*storage.ComplianceOperatorProfileV2, 0)
			for _, profileName := range tc.profileNames {
				testProfiles = append(testProfiles, getTestProfile(profileName, "1.0.0", "platform", "ocp4", tc.newClusterIDs[0], 1))
			}
			suite.profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Return(testProfiles, nil).Times(1)
			suite.scanConfigDS.EXPECT().UpsertScanConfiguration(gomock.Any(), gomock.Any()).Return(nil).Times(1)

			// Remove cluster calls.
			suite.connectionMgr.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(nil).Times(len(expectedRemovedClusters))
			suite.scanConfigDS.EXPECT().RemoveClusterStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(len(expectedRemovedClusters))

			// Update cluster calls.
			suite.connectionMgr.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(nil).Times(len(tc.newClusterIDs))
			suite.clusterDatastore.EXPECT().GetClusterName(gomock.Any(), gomock.Any()).Return("test_cluster", true, nil).Times(len(tc.newClusterIDs))
			suite.scanConfigDS.EXPECT().UpdateClusterStatus(gomock.Any(), gomock.Any(), gomock.Any(), "", "test_cluster").Return(nil).Times(len(tc.newClusterIDs))

			// Testing removeObsoleteResultsByClusters call.
			if len(expectedRemovedClusters) > 0 {
				suite.resultsDS.EXPECT().DeleteResultsByScanConfigAndCluster(suite.hasWriteCtx, oldScanConfig.GetScanConfigName(), gomock.Cond(func(clusterIDs []string) bool {
					return suite.ElementsMatch(expectedRemovedClusters, clusterIDs)
				})).Return(nil).Times(1)
			}
			_, err := suite.manager.UpdateScanRequest(suite.hasWriteCtx, newScanConfig, tc.newClusterIDs)
			suite.NoError(err)
		})
	}
}

func (suite *complianceManagerTestSuite) TestRemoveObsoleteResultsByProfiles() {
	cases := map[string]struct {
		clusterIDs      []string
		oldProfileNames []string
		newProfileNames []string
	}{
		"no changes": {
			clusterIDs:      []string{testconsts.Cluster1},
			oldProfileNames: []string{"ocp4-cis"},
			newProfileNames: []string{"ocp4-cis"},
		},
		"new profile": {
			clusterIDs:      []string{testconsts.Cluster1, testconsts.Cluster2},
			oldProfileNames: []string{"ocp4-cis"},
			newProfileNames: []string{"ocp4-cis", "ocp4-cis-node"},
		},
		"remove profile": {
			clusterIDs:      []string{testconsts.Cluster2},
			oldProfileNames: []string{"ocp4-cis", "ocp4-cis-node"},
			newProfileNames: []string{"ocp4-cis"},
		},
	}

	for testName, tc := range cases {
		suite.T().Run(testName, func(tt *testing.T) {
			oldScanConfig := getTestRecWithClustersAndProfiles(mockScanID, tc.clusterIDs, tc.oldProfileNames)
			newScanConfig := getTestRecWithClustersAndProfiles(mockScanID, tc.clusterIDs, tc.newProfileNames)

			expectedRuleRefIDs := make([]string, 0)
			newTestProfiles := make([]*storage.ComplianceOperatorProfileV2, 0)
			for _, profileName := range tc.newProfileNames {
				testProfile := getTestProfile(profileName, "1.0.0", "platform", "ocp4", tc.clusterIDs[0], 1)
				newTestProfiles = append(newTestProfiles, testProfile)

				for _, clusterID := range tc.clusterIDs {
					for _, rule := range testProfile.GetRules() {
						expectedRuleRefIDs = append(expectedRuleRefIDs, internaltov2storage.BuildNameRefID(clusterID, rule.GetRuleName()))
					}
				}
			}

			suite.scanConfigDS.EXPECT().GetScanConfiguration(gomock.Any(), mockScanID).Return(oldScanConfig, true, nil).Times(1)
			suite.scanConfigDS.EXPECT().ScanConfigurationProfileExists(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
			suite.profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Return(newTestProfiles, nil).Times(1)
			suite.scanConfigDS.EXPECT().UpsertScanConfiguration(gomock.Any(), gomock.Any()).Return(nil).Times(1)

			suite.connectionMgr.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(nil).Times(len(tc.clusterIDs))
			suite.clusterDatastore.EXPECT().GetClusterName(gomock.Any(), gomock.Any()).Return("test_cluster", true, nil).Times(len(tc.clusterIDs))
			suite.scanConfigDS.EXPECT().UpdateClusterStatus(gomock.Any(), gomock.Any(), gomock.Any(), "", "test_cluster").Return(nil).Times(len(tc.clusterIDs))

			expectedRemovedProfileNames := set.NewStringSet(tc.oldProfileNames...).Difference(set.NewStringSet(tc.newProfileNames...)).AsSlice()
			if len(expectedRemovedProfileNames) > 0 {
				suite.profileDS.EXPECT().SearchProfiles(gomock.Any(), gomock.Any()).Return(newTestProfiles, nil).Times(1)
				suite.resultsDS.EXPECT().DeleteResultsByScanConfigAndRules(suite.hasWriteCtx, oldScanConfig.GetScanConfigName(), gomock.Cond(func(ruleRefIds []string) bool {
					return suite.ElementsMatch(expectedRuleRefIDs, ruleRefIds)
				})).Return(nil).Times(1)
			}
			_, err := suite.manager.UpdateScanRequest(suite.hasWriteCtx, newScanConfig, tc.clusterIDs)
			suite.NoError(err)
		})
	}
}
