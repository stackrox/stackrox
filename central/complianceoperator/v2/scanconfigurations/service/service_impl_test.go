package service

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	blobDSMocks "github.com/stackrox/rox/central/blob/datastore/mocks"
	clusterDatastoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	benchmarkMocks "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/datastore/mocks"
	managerMocks "github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager/mocks"
	profileDatastore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/mocks"
	"github.com/stackrox/rox/central/complianceoperator/v2/report"
	snapshotMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/datastore/mocks"
	reportManagerMocks "github.com/stackrox/rox/central/complianceoperator/v2/report/manager/mocks"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	scanSettingBindingMocks "github.com/stackrox/rox/central/complianceoperator/v2/scansettingbindings/datastore/mocks"
	suiteMocks "github.com/stackrox/rox/central/complianceoperator/v2/suites/datastore/mocks"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore/mocks"
	"github.com/stackrox/rox/central/reports/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	mockClusterName = "mock-cluster"
)

var (
	defaultStorageSchedule = storage.Schedule_builder{
		IntervalType: 2,
		Hour:         15,
		Minute:       0,
		DaysOfWeek: storage.Schedule_DaysOfWeek_builder{
			Days: []int32{1, 2, 3, 4, 5, 6, 7},
		}.Build(),
	}.Build()

	defaultAPISchedule = apiV2.Schedule_builder{
		IntervalType: 1,
		Hour:         15,
		Minute:       0,
		DaysOfWeek: apiV2.Schedule_DaysOfWeek_builder{
			Days: []int32{1, 2, 3, 4, 5, 6, 7},
		}.Build(),
	}.Build()

	apiRequester = &apiV2.SlimUser{
		Id:   "uid",
		Name: "name",
	}

	storageRequester = &storage.SlimUser{
		Id:   "uid",
		Name: "name",
	}
)

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, &serviceImpl{})
}

func TestComplianceScanConfigService(t *testing.T) {
	suite.Run(t, new(ComplianceScanConfigServiceTestSuite))
}

type ComplianceScanConfigServiceTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx                         context.Context
	manager                     *managerMocks.MockManager
	reportManager               *reportManagerMocks.MockManager
	scanConfigDatastore         *scanConfigMocks.MockDataStore
	scanSettingBindingDatastore *scanSettingBindingMocks.MockDataStore
	suiteDataStore              *suiteMocks.MockDataStore
	notifierDS                  *notifierDS.MockDataStore
	profileDS                   *profileDatastore.MockDataStore
	clusterDatastore            *clusterDatastoreMocks.MockDataStore
	benchmarkDS                 *benchmarkMocks.MockDataStore
	snapshotDS                  *snapshotMocks.MockDataStore
	blobDS                      *blobDSMocks.MockDatastore
	service                     Service
}

func (s *ComplianceScanConfigServiceTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip test when compliance enhancements are disabled")
		s.T().SkipNow()
	}

	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *ComplianceScanConfigServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.manager = managerMocks.NewMockManager(s.mockCtrl)
	s.reportManager = reportManagerMocks.NewMockManager(s.mockCtrl)
	s.scanConfigDatastore = scanConfigMocks.NewMockDataStore(s.mockCtrl)
	s.scanSettingBindingDatastore = scanSettingBindingMocks.NewMockDataStore(s.mockCtrl)
	s.suiteDataStore = suiteMocks.NewMockDataStore(s.mockCtrl)
	s.profileDS = profileDatastore.NewMockDataStore(s.mockCtrl)
	s.clusterDatastore = clusterDatastoreMocks.NewMockDataStore(s.mockCtrl)
	s.benchmarkDS = benchmarkMocks.NewMockDataStore(s.mockCtrl)
	s.snapshotDS = snapshotMocks.NewMockDataStore(s.mockCtrl)
	s.blobDS = blobDSMocks.NewMockDatastore(s.mockCtrl)
	s.service = New(s.scanConfigDatastore, s.scanSettingBindingDatastore, s.suiteDataStore, s.manager, s.reportManager, s.notifierDS, s.profileDS, s.benchmarkDS, s.clusterDatastore, s.snapshotDS, s.blobDS)
}

func (s *ComplianceScanConfigServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *ComplianceScanConfigServiceTestSuite) TestComplianceScanConfigurationName() {
	allAccessContext := sac.WithAllAccess(context.Background())

	request := getTestAPIRec()
	request.SetScanName("test@scan")
	request.SetId(uuid.NewDummy().String())
	processResponse := convertV2ScanConfigToStorage(allAccessContext, request)
	processResponse.SetId(uuid.NewDummy().String())

	_, err := s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)

	request.SetScanName("testscan_")
	_, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)

	request.SetScanName("default")
	_, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Contains(err.Error(), "Scan configuration name \"default\" cannot be used as it is reserved by the Compliance Operator")

	request.SetScanName("default-auto-apply")
	_, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Contains(err.Error(), "Scan configuration name \"default-auto-apply\" cannot be used as it is reserved by the Compliance Operator")
}

func (s *ComplianceScanConfigServiceTestSuite) TestCreateComplianceScanConfiguration() {
	allAccessContext := sac.WithAllAccess(context.Background())

	request := getTestAPIRec()
	request.SetId(uuid.NewDummy().String())
	storageRequest := convertV2ScanConfigToStorage(allAccessContext, request)
	processResponse := convertV2ScanConfigToStorage(allAccessContext, request)
	processResponse.SetId(uuid.NewDummy().String())
	s.manager.EXPECT().ProcessScanRequest(gomock.Any(), storageRequest, []string{fixtureconsts.Cluster1}).Return(processResponse, nil).Times(1)
	cocscs := &storage.ComplianceOperatorClusterScanConfigStatus{}
	cocscs.SetClusterId(fixtureconsts.Cluster1)
	cocscs.SetScanConfigId(uuid.NewDummy().String())
	cocscs.SetErrors([]string{"Error 1", "Error 2", "Error 3"})
	s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(allAccessContext, uuid.NewDummy().String()).Return([]*storage.ComplianceOperatorClusterScanConfigStatus{
		cocscs,
	}, nil).Times(1)

	config, err := s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().NoError(err)
	// ID will be added to the record and returned.  Add it to the validation object
	request.SetId(uuid.NewDummy().String())
	protoassert.Equal(s.T(), request, config)

	// reset for error testing
	request = getTestAPIRec()
	request.ClearScanConfig()
	config, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "The scan configuration is nil.")
	s.Require().Nil(config)

	request = getTestAPIRec()
	request.SetClusters([]string{})
	config, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "At least one cluster is required for a scan configuration")
	s.Require().Nil(config)

	request = getTestAPIRec()
	request.GetScanConfig().SetProfiles([]string{})
	config, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "At least one profile is required for a scan configuration")
	s.Require().Nil(config)
}

func (s *ComplianceScanConfigServiceTestSuite) TestUpdateComplianceScanConfiguration() {
	allAccessContext := sac.WithAllAccess(context.Background())

	request := getTestAPIRec()
	request.SetId(uuid.NewDummy().String())
	storageRequest := convertV2ScanConfigToStorage(allAccessContext, request)
	processResponse := convertV2ScanConfigToStorage(allAccessContext, request)
	processResponse.SetId(uuid.NewDummy().String())
	s.manager.EXPECT().UpdateScanRequest(gomock.Any(), storageRequest, []string{fixtureconsts.Cluster1}).Return(processResponse, nil).Times(1)

	_, err := s.service.UpdateComplianceScanConfiguration(allAccessContext, request)
	s.Require().NoError(err)

	// Test Case 2: Update with Empty ID
	request.SetId("")
	_, err = s.service.UpdateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "Scan configuration ID is required: invalid arguments")

	// Test Case 3: No ScanConfig
	request = getTestAPIRec()
	request.SetId(uuid.NewDummy().String())
	request.ClearScanConfig()
	_, err = s.service.UpdateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "The scan configuration is nil.")

	// Test Case 4: No clusters
	request = getTestAPIRec()
	request.SetId(uuid.NewDummy().String())
	request.SetClusters([]string{})
	_, err = s.service.UpdateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "At least one cluster is required for a scan configuration")

	// Test Case 5: No profiles
	request = getTestAPIRec()
	request.SetId(uuid.NewDummy().String())
	request.GetScanConfig().SetProfiles([]string{})
	_, err = s.service.UpdateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "At least one profile is required for a scan configuration")
}

func (s *ComplianceScanConfigServiceTestSuite) TestDeleteComplianceScanConfiguration() {
	allAccessContext := sac.WithAllAccess(context.Background())

	// Test Case 1: Successful Deletion
	validID := "validScanConfigID"
	snapshotID := "snapshot-1"
	snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
		getSnapshot(snapshotID, storageRequester),
	}
	s.manager.EXPECT().DeleteScan(gomock.Any(), validID).Return(nil).Times(1)
	s.snapshotDS.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).Times(1).Return(snapshots, nil)
	s.blobDS.EXPECT().Delete(gomock.Cond[context.Context](func(ctx context.Context) bool {
		return validateBlobContext(ctx, storage.Access_READ_WRITE_ACCESS)
	}), common.GetComplianceReportBlobPath(validID, snapshotID)).Times(1).Return(nil)

	rbid := &v2.ResourceByID{}
	rbid.SetId(validID)
	_, err := s.service.DeleteComplianceScanConfiguration(allAccessContext, rbid)
	s.Require().NoError(err)

	// Test Case 2: Deletion with Empty ID
	rbid2 := &v2.ResourceByID{}
	rbid2.SetId("")
	_, err = s.service.DeleteComplianceScanConfiguration(allAccessContext, rbid2)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "Scan configuration ID is required for deletion")

	// Test Case 3: Deletion Fails in Manager
	failingID := "failingScanConfigID"
	s.manager.EXPECT().DeleteScan(gomock.Any(), failingID).Return(errors.New("manager error")).Times(1)
	s.snapshotDS.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

	rbid3 := &v2.ResourceByID{}
	rbid3.SetId(failingID)
	_, err = s.service.DeleteComplianceScanConfiguration(allAccessContext, rbid3)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "Unable to delete scan config")
}

func (s *ComplianceScanConfigServiceTestSuite) TestCreateComplianceScanConfigurationScanExists() {
	allAccessContext := sac.WithAllAccess(context.Background())

	request := getTestAPIRec()
	storageRequest := convertV2ScanConfigToStorage(allAccessContext, request)
	managerErr := errors.Errorf("Scan Configuration named %q already exists.", request.GetScanName())
	s.manager.EXPECT().ProcessScanRequest(gomock.Any(), storageRequest, []string{fixtureconsts.Cluster1}).Return(nil, managerErr).Times(1)
	expectedErr := errors.Wrapf(errox.InvalidArgs, "Unable to process scan config. Scan Configuration named %q already exists.", request.GetScanName())

	config, err := s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Equal(expectedErr.Error(), err.Error())
	s.Require().Nil(config)
}

func (s *ComplianceScanConfigServiceTestSuite) TestListComplianceScanConfigurations() {
	allAccessContext := sac.WithAllAccess(context.Background())
	createdTime := timestamp.Now().GoTime()
	lastUpdatedTime := timestamp.Now().GoTime()

	testCases := []struct {
		desc           string
		query          *apiV2.RawQuery
		expectedQ      *v1.Query
		expectedCountQ *v1.Query
	}{
		{
			desc:           "Empty query",
			query:          apiV2.RawQuery_builder{Query: ""}.Build(),
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
		},
		{
			desc:  "Query with search field",
			query: apiV2.RawQuery_builder{Query: "Cluster ID:id"}.Build(),
			expectedQ: search.NewQueryBuilder().AddStrings(search.ClusterID, "id").
				WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.NewQueryBuilder().AddStrings(search.ClusterID, "id").ProtoQuery(),
		},
		{
			desc: "Query with custom pagination",
			query: apiV2.RawQuery_builder{
				Query:      "",
				Pagination: apiV2.Pagination_builder{Limit: 1}.Build(),
			}.Build(),
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(1)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			expectedResp := &apiV2.ListComplianceScanConfigurationsResponse{}
			expectedResp.SetConfigurations([]*apiV2.ComplianceScanConfigurationStatus{
				getTestAPIStatusRec(createdTime, lastUpdatedTime),
			})
			expectedResp.SetTotalCount(6)

			s.scanConfigDatastore.EXPECT().GetScanConfigurations(allAccessContext, tc.expectedQ).
				Return([]*storage.ComplianceOperatorScanConfigurationV2{
					storage.ComplianceOperatorScanConfigurationV2_builder{
						Id:                     uuid.NewDummy().String(),
						ScanConfigName:         "test-scan",
						AutoApplyRemediations:  false,
						AutoUpdateRemediations: false,
						OneTimeScan:            false,
						Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
							storage.ComplianceOperatorScanConfigurationV2_ProfileName_builder{
								ProfileName: "ocp4-cis",
							}.Build(),
						},
						StrictNodeScan:  false,
						Schedule:        defaultStorageSchedule,
						CreatedTime:     protoconv.ConvertTimeToTimestamp(createdTime),
						LastUpdatedTime: protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
						ModifiedBy:      storageRequester,
						Description:     "test-description",
						Notifiers:       []*storage.NotifierConfiguration{},
					}.Build(),
				}, nil).Times(1)

			cocscs := &storage.ComplianceOperatorClusterScanConfigStatus{}
			cocscs.SetClusterId(fixtureconsts.Cluster1)
			cocscs.SetClusterName(mockClusterName)
			cocscs.SetScanConfigId(uuid.NewDummy().String())
			cocscs.SetErrors([]string{"Error 1", "Error 2", "Error 3"})
			s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(allAccessContext, uuid.NewDummy().String()).Return([]*storage.ComplianceOperatorClusterScanConfigStatus{
				cocscs,
			}, nil).Times(1)

			s.scanSettingBindingDatastore.EXPECT().GetScanSettingBindings(allAccessContext, gomock.Any()).Return([]*storage.ComplianceOperatorScanSettingBindingV2{
				storage.ComplianceOperatorScanSettingBindingV2_builder{
					ClusterId: fixtureconsts.Cluster1,
					Status: storage.ComplianceOperatorStatus_builder{
						Phase: "READY",
						Conditions: []*storage.ComplianceOperatorCondition{
							storage.ComplianceOperatorCondition_builder{
								Type:    "READY",
								Status:  "False",
								Message: "This binding is not ready",
							}.Build(),
						},
					}.Build(),
				}.Build(),
			}, nil).Times(1)
			s.suiteDataStore.EXPECT().GetSuites(allAccessContext, gomock.Any()).Return([]*storage.ComplianceOperatorSuiteV2{
				storage.ComplianceOperatorSuiteV2_builder{
					Id:        uuid.NewDummy().String(),
					ClusterId: fixtureconsts.Cluster1,
					Status: storage.ComplianceOperatorStatus_builder{
						Phase:  "DONE",
						Result: "NON-COMPLIANT",
						Conditions: []*storage.ComplianceOperatorCondition{
							storage.ComplianceOperatorCondition_builder{
								Type:               "Processing",
								Status:             "False",
								LastTransitionTime: protocompat.GetProtoTimestampFromSeconds(lastUpdatedTime.UTC().Unix() - 10),
							}.Build(),
							storage.ComplianceOperatorCondition_builder{
								Type:               "Ready",
								Status:             "True",
								LastTransitionTime: protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
							}.Build(),
						},
					}.Build(),
				}.Build(),
			}, nil).Times(1)

			s.scanConfigDatastore.EXPECT().CountScanConfigurations(allAccessContext, tc.expectedCountQ).
				Return(6, nil).Times(1)

			configs, err := s.service.ListComplianceScanConfigurations(allAccessContext, tc.query)
			s.Require().NoError(err)
			protoassert.Equal(s.T(), expectedResp, configs)
		})
	}
}

func (s *ComplianceScanConfigServiceTestSuite) TestGetComplianceScanConfiguration() {
	allAccessContext := sac.WithAllAccess(context.Background())
	createdTime := timestamp.Now().GoTime()
	lastUpdatedTime := timestamp.Now().GoTime()

	testCases := []struct {
		desc         string
		scanID       string
		expectedResp *apiV2.ComplianceScanConfigurationStatus
		expectedErr  error
		found        bool
	}{
		{
			desc:         "Valid ID with a config",
			scanID:       uuid.NewDummy().String(),
			expectedResp: getTestAPIStatusRec(createdTime, lastUpdatedTime),
			found:        true,
			expectedErr:  nil,
		},
		{
			desc:         "ID represents no config",
			scanID:       "bad id",
			expectedResp: nil,
			found:        false,
			expectedErr:  errors.New("failed to retrieve compliance scan configuration with id \"bad id\".: record not found"),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			if tc.found {
				cp := &storage.ComplianceOperatorScanConfigurationV2_ProfileName{}
				cp.SetProfileName("ocp4-cis")
				coscv2 := &storage.ComplianceOperatorScanConfigurationV2{}
				coscv2.SetId(uuid.NewDummy().String())
				coscv2.SetScanConfigName("test-scan")
				coscv2.SetAutoApplyRemediations(false)
				coscv2.SetAutoUpdateRemediations(false)
				coscv2.SetOneTimeScan(false)
				coscv2.SetProfiles([]*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
					cp,
				})
				coscv2.SetStrictNodeScan(false)
				coscv2.SetSchedule(defaultStorageSchedule)
				coscv2.SetCreatedTime(protoconv.ConvertTimeToTimestamp(createdTime))
				coscv2.SetLastUpdatedTime(protoconv.ConvertTimeToTimestamp(lastUpdatedTime))
				coscv2.SetModifiedBy(storageRequester)
				coscv2.SetDescription("test-description")
				coscv2.SetNotifiers([]*storage.NotifierConfiguration{})
				s.scanConfigDatastore.EXPECT().GetScanConfiguration(allAccessContext, tc.scanID).
					Return(coscv2, true, nil).Times(1)

				s.suiteDataStore.EXPECT().GetSuites(allAccessContext, gomock.Any()).Return([]*storage.ComplianceOperatorSuiteV2{
					storage.ComplianceOperatorSuiteV2_builder{
						Id:        uuid.NewDummy().String(),
						ClusterId: fixtureconsts.Cluster1,
						Name:      "test-scan",
						Status: storage.ComplianceOperatorStatus_builder{
							Phase:  "DONE",
							Result: "NON-COMPLIANT",
							Conditions: []*storage.ComplianceOperatorCondition{
								storage.ComplianceOperatorCondition_builder{
									Type:               "Ready",
									Status:             "True",
									LastTransitionTime: protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
								}.Build(),
								storage.ComplianceOperatorCondition_builder{
									Type:               "Processing",
									Status:             "False",
									LastTransitionTime: protocompat.GetProtoTimestampFromSeconds(lastUpdatedTime.UTC().Unix() - 10),
								}.Build(),
							},
						}.Build(),
					}.Build(),
				}, nil).Times(1)

				cocscs := &storage.ComplianceOperatorClusterScanConfigStatus{}
				cocscs.SetClusterId(fixtureconsts.Cluster1)
				cocscs.SetClusterName(mockClusterName)
				cocscs.SetScanConfigId(uuid.NewDummy().String())
				cocscs.SetErrors([]string{"Error 1", "Error 2", "Error 3"})
				s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(allAccessContext, uuid.NewDummy().String()).Return([]*storage.ComplianceOperatorClusterScanConfigStatus{
					cocscs,
				}, nil).Times(1)

				s.scanSettingBindingDatastore.EXPECT().GetScanSettingBindings(allAccessContext, gomock.Any()).Return([]*storage.ComplianceOperatorScanSettingBindingV2{
					storage.ComplianceOperatorScanSettingBindingV2_builder{
						ClusterId: fixtureconsts.Cluster1,
						Status: storage.ComplianceOperatorStatus_builder{
							Phase: "READY",
							Conditions: []*storage.ComplianceOperatorCondition{
								storage.ComplianceOperatorCondition_builder{
									Type:    "READY",
									Status:  "False",
									Message: "This binding is not ready",
								}.Build(),
							},
						}.Build(),
					}.Build(),
				}, nil).Times(1)
			} else {
				s.scanConfigDatastore.EXPECT().GetScanConfiguration(allAccessContext, tc.scanID).
					Return(nil, false, errors.New("record not found")).Times(1)
			}

			rbid := &apiV2.ResourceByID{}
			rbid.SetId(tc.scanID)
			config, err := s.service.GetComplianceScanConfiguration(allAccessContext, rbid)
			if tc.expectedErr == nil {
				s.Require().NoError(err)
			} else {
				s.Require().Equal(tc.expectedErr.Error(), err.Error())
			}
			protoassert.Equal(s.T(), tc.expectedResp, config)
		})
	}
}

func (s *ComplianceScanConfigServiceTestSuite) ListComplianceScanConfigProfiles() {
	allAccessContext := sac.WithAllAccess(context.Background())

	testCases := []struct {
		desc           string
		query          *apiV2.RawQuery
		expectedQ      *v1.Query
		expectedCountQ *v1.Query
	}{
		{
			desc:           "Empty query",
			query:          apiV2.RawQuery_builder{Query: ""}.Build(),
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
		},
		{
			desc:  "Query with search field",
			query: apiV2.RawQuery_builder{Query: "Cluster ID:id"}.Build(),
			expectedQ: search.NewQueryBuilder().AddStrings(search.ClusterID, "id").
				WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.NewQueryBuilder().AddStrings(search.ClusterID, "id").ProtoQuery(),
		},
		{
			desc: "Query with custom pagination",
			query: apiV2.RawQuery_builder{
				Query:      "",
				Pagination: apiV2.Pagination_builder{Limit: 1}.Build(),
			}.Build(),
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(1)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			expectedResp := &apiV2.ListComplianceScanConfigsProfileResponse{}
			expectedResp.SetProfiles(nil)
			expectedResp.SetTotalCount(6)

			s.scanConfigDatastore.EXPECT().GetProfilesNames(gomock.Any(), tc.query).Return([]string{"ocp4"}, nil).Times(1)
			s.scanConfigDatastore.EXPECT().DistinctProfiles(gomock.Any(), tc.expectedCountQ).Return(map[string]int{"ocp4": 1}, nil).Times(1)

			searchQuery := search.NewQueryBuilder().AddSelectFields().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").ProtoQuery()
			searchQuery.SetPagination(&v1.QueryPagination{})

			profiles := []*storage.ComplianceOperatorProfileV2{
				storage.ComplianceOperatorProfileV2_builder{
					Name:           "ocp4",
					ProductType:    "platform",
					Description:    "this is a test",
					Title:          "A Title",
					ProfileVersion: "version 1",
					Rules: []*storage.ComplianceOperatorProfileV2_Rule{
						storage.ComplianceOperatorProfileV2_Rule_builder{
							RuleName: "test 1",
						}.Build(),
						storage.ComplianceOperatorProfileV2_Rule_builder{
							RuleName: "test 2",
						}.Build(),
						storage.ComplianceOperatorProfileV2_Rule_builder{
							RuleName: "test 3",
						}.Build(),
						storage.ComplianceOperatorProfileV2_Rule_builder{
							RuleName: "test 4",
						}.Build(),
						storage.ComplianceOperatorProfileV2_Rule_builder{
							RuleName: "test 5",
						}.Build(),
					},
				}.Build(),
			}
			s.profileDS.EXPECT().SearchProfiles(gomock.Any(), searchQuery).Return(profiles, nil).Times(1)

			for _, profile := range profiles {
				cobv2 := &storage.ComplianceOperatorBenchmarkV2{}
				cobv2.SetId(uuid.NewV4().String())
				cobv2.SetName("CIS")
				cobv2.SetShortName("OCP_CIS")
				cobv2.SetVersion("1-5")
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(s.ctx, profile.GetName()).Return([]*storage.ComplianceOperatorBenchmarkV2{cobv2}, nil).Times(1)
			}

			configProfiles, err := s.service.ListComplianceScanConfigProfiles(allAccessContext, tc.query)
			s.Require().NoError(err)
			protoassert.Equal(s.T(), expectedResp, configProfiles)
		})
	}
}

func (s *ComplianceScanConfigServiceTestSuite) TestRunComplianceScanConfigurationWithValidScanConfigIdSucceeds() {
	allAccessContext := sac.WithAllAccess(context.Background())

	validID := "validScanConfigID"
	s.manager.EXPECT().ProcessRescanRequest(gomock.Any(), validID).Return(nil).Times(1)

	rbid := &v2.ResourceByID{}
	rbid.SetId(validID)
	_, err := s.service.RunComplianceScanConfiguration(allAccessContext, rbid)
	s.Require().NoError(err)
}

func (s *ComplianceScanConfigServiceTestSuite) TestRunComplianceScanConfigurationWithInvalidScanConfigIdFails() {
	allAccessContext := sac.WithAllAccess(context.Background())

	invalidID := ""
	rbid := &v2.ResourceByID{}
	rbid.SetId(invalidID)
	_, err := s.service.RunComplianceScanConfiguration(allAccessContext, rbid)
	s.Require().Error(err)
}

func (s *ComplianceScanConfigServiceTestSuite) TestRunReport() {
	s.T().Setenv(features.ComplianceReporting.EnvVar(), "true")
	if !features.ComplianceReporting.Enabled() {
		s.T().Skip("Skip test when compliance reporting feature flag is disabled")
		s.T().SkipNow()
	}

	allAccessContext := sac.WithAllAccess(context.Background())

	user := &storage.SlimUser{}
	user.SetId("user-1")
	user.SetName("user-1")

	ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, user)

	invalidID := ""
	crrr := &v2.ComplianceRunReportRequest{}
	crrr.SetScanConfigId(invalidID)
	_, err := s.service.RunReport(ctx, crrr)
	s.Require().Error(err)

	nonExistentScanConfigID := "does-not-exist-scan-config-1"
	s.scanConfigDatastore.EXPECT().GetScanConfiguration(ctx, nonExistentScanConfigID).Return(nil, false, nil)
	crrr2 := &v2.ComplianceRunReportRequest{}
	crrr2.SetScanConfigId(nonExistentScanConfigID)
	_, err = s.service.RunReport(ctx, crrr2)
	s.Require().Error(err)

	validScanConfigID := "scan-config-1"
	validScanConfig := &storage.ComplianceOperatorScanConfigurationV2{}
	validScanConfig.SetId("scan-config-1")
	validScanConfig.SetScanConfigName("scan-config-1")
	s.scanConfigDatastore.EXPECT().GetScanConfiguration(ctx, validScanConfigID).Return(validScanConfig, true, nil)
	s.reportManager.EXPECT().SubmitReportRequest(ctx, validScanConfig, storage.ComplianceOperatorReportStatus_EMAIL).Return(nil)

	crrr3 := &v2.ComplianceRunReportRequest{}
	crrr3.SetScanConfigId(validScanConfigID)
	crrr3.SetReportNotificationMethod(v2.NotificationMethod_EMAIL)
	resp, err := s.service.RunReport(ctx, crrr3)
	s.Require().NoError(err)
	s.Equal(v2.ComplianceRunReportResponse_SUBMITTED, resp.GetRunState(), "Failed to submit report")

	s.scanConfigDatastore.EXPECT().GetScanConfiguration(ctx, validScanConfigID).Return(validScanConfig, true, nil)
	s.reportManager.EXPECT().SubmitReportRequest(ctx, validScanConfig, storage.ComplianceOperatorReportStatus_DOWNLOAD).Return(nil)

	crrr4 := &v2.ComplianceRunReportRequest{}
	crrr4.SetScanConfigId(validScanConfigID)
	crrr4.SetReportNotificationMethod(v2.NotificationMethod_DOWNLOAD)
	resp, err = s.service.RunReport(ctx, crrr4)
	s.Require().NoError(err)
	s.Equal(v2.ComplianceRunReportResponse_SUBMITTED, resp.GetRunState(), "Failed to submit report")
}

func (s *ComplianceScanConfigServiceTestSuite) TestGetReportHistory() {
	s.T().Setenv(features.ComplianceReporting.EnvVar(), "true")
	s.T().Setenv(features.ScanScheduleReportJobs.EnvVar(), "true")
	if !features.ComplianceReporting.Enabled() || !features.ScanScheduleReportJobs.Enabled() {
		s.T().Skipf("compliance reporting feature flag is disabled")
		s.T().SkipNow()
	}

	allAccessContext := sac.WithAllAccess(context.Background())

	s.Run("Invalid ID", func() {
		invalidID := ""
		crhr := &v2.ComplianceReportHistoryRequest{}
		crhr.SetId(invalidID)
		_, err := s.service.GetReportHistory(allAccessContext, crhr)
		s.Require().Error(err)
	})

	s.Run("Snapshot search error", func() {
		scanConfigID := "scan-config-1"

		s.snapshotDS.EXPECT().SearchSnapshots(allAccessContext, gomock.Any()).Return(nil, errors.New("some error"))

		crhr := &v2.ComplianceReportHistoryRequest{}
		crhr.SetId(scanConfigID)
		_, err := s.service.GetReportHistory(allAccessContext, crhr)
		s.Require().Error(err)
	})

	s.Run("Success", func() {
		scanConfigID := "scan-config-1"
		now := protocompat.TimestampNow()
		snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
			storage.ComplianceOperatorReportSnapshotV2_builder{
				ReportId:            "snapshot-1",
				ScanConfigurationId: scanConfigID,
				ReportStatus: storage.ComplianceOperatorReportStatus_builder{
					ReportRequestType:        storage.ComplianceOperatorReportStatus_SCHEDULED,
					ReportNotificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
					StartedAt:                now,
					CompletedAt:              now,
				}.Build(),
				FailedClusters: []*storage.ComplianceOperatorReportSnapshotV2_FailedCluster{
					storage.ComplianceOperatorReportSnapshotV2_FailedCluster_builder{
						ClusterId:       "cluster-1",
						ClusterName:     "cluster-1",
						OperatorVersion: "v1.6.0",
						Reasons:         []string{report.INTERNAL_ERROR},
					}.Build(),
				},
			}.Build(),
		}
		sc := &storage.ComplianceOperatorScanConfigurationV2{}
		sc.SetId(scanConfigID)
		sc.SetScanConfigName(scanConfigID)

		s.snapshotDS.EXPECT().SearchSnapshots(allAccessContext, gomock.Any()).Return(snapshots, nil)
		s.scanConfigDatastore.EXPECT().GetScanConfiguration(allAccessContext, scanConfigID).Return(sc, true, nil)
		s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(allAccessContext, scanConfigID).Return(nil, nil)
		s.suiteDataStore.EXPECT().GetSuites(allAccessContext, gomock.Any()).Return(nil, nil)

		crhr := &v2.ComplianceReportHistoryRequest{}
		crhr.SetId(scanConfigID)
		res, err := s.service.GetReportHistory(allAccessContext, crhr)
		s.Require().NoError(err)
		protoassert.Equal(s.T(), v2.ComplianceReportHistoryResponse_builder{
			ComplianceReportSnapshots: []*v2.ComplianceReportSnapshot{
				v2.ComplianceReportSnapshot_builder{
					ReportJobId:  "snapshot-1",
					ScanConfigId: scanConfigID,
					ReportStatus: v2.ComplianceReportStatus_builder{
						ReportRequestType:        v2.ComplianceReportStatus_SCHEDULED,
						ReportNotificationMethod: v2.NotificationMethod_EMAIL,
						StartedAt:                now,
						CompletedAt:              now,
						FailedClusters: []*v2.FailedCluster{
							v2.FailedCluster_builder{
								ClusterId:       "cluster-1",
								ClusterName:     "cluster-1",
								OperatorVersion: "v1.6.0",
								Reason:          report.INTERNAL_ERROR,
							}.Build(),
						},
					}.Build(),
					ReportData: v2.ComplianceScanConfigurationStatus_builder{
						Id:       scanConfigID,
						ScanName: scanConfigID,
						ScanConfig: v2.BaseComplianceScanConfigurationSettings_builder{
							OneTimeScan: false,
							Profiles:    []string{},
							Notifiers:   []*v2.NotifierConfiguration{},
						}.Build(),
						ClusterStatus: []*v2.ClusterScanStatus{},
						ModifiedBy:    &v2.SlimUser{},
					}.Build(),
					User:                &v2.SlimUser{},
					IsDownloadAvailable: false,
				}.Build(),
			},
		}.Build(), res)
	})

	s.Run("Success for download", func() {
		scanConfigID := "scan-config-1"
		now := protocompat.TimestampNow()
		cors := &storage.ComplianceOperatorReportStatus{}
		cors.SetReportRequestType(storage.ComplianceOperatorReportStatus_ON_DEMAND)
		cors.SetReportNotificationMethod(storage.ComplianceOperatorReportStatus_DOWNLOAD)
		cors.SetStartedAt(now)
		cors.SetCompletedAt(now)
		cors.SetRunState(storage.ComplianceOperatorReportStatus_GENERATED)
		corsv2 := &storage.ComplianceOperatorReportSnapshotV2{}
		corsv2.SetReportId("snapshot-1")
		corsv2.SetScanConfigurationId(scanConfigID)
		corsv2.SetReportStatus(cors)
		snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
			corsv2,
		}
		sc := &storage.ComplianceOperatorScanConfigurationV2{}
		sc.SetId(scanConfigID)
		sc.SetScanConfigName(scanConfigID)

		s.snapshotDS.EXPECT().SearchSnapshots(allAccessContext, gomock.Any()).Return(snapshots, nil)
		s.scanConfigDatastore.EXPECT().GetScanConfiguration(allAccessContext, scanConfigID).Return(sc, true, nil)
		s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(allAccessContext, scanConfigID).Return(nil, nil)
		s.suiteDataStore.EXPECT().GetSuites(allAccessContext, gomock.Any()).Return(nil, nil)
		s.blobDS.EXPECT().Search(gomock.Cond[context.Context](func(ctx context.Context) bool {
			return validateBlobContext(ctx, storage.Access_READ_ACCESS)
		}), gomock.Any()).Times(1).Return([]search.Result{
			{
				ID: common.GetComplianceReportBlobPath(scanConfigID, "snapshot-1"),
			},
		}, nil)

		crhr := &v2.ComplianceReportHistoryRequest{}
		crhr.SetId(scanConfigID)
		res, err := s.service.GetReportHistory(allAccessContext, crhr)
		s.Require().NoError(err)
		protoassert.Equal(s.T(), v2.ComplianceReportHistoryResponse_builder{
			ComplianceReportSnapshots: []*v2.ComplianceReportSnapshot{
				v2.ComplianceReportSnapshot_builder{
					ReportJobId:  "snapshot-1",
					ScanConfigId: scanConfigID,
					ReportStatus: v2.ComplianceReportStatus_builder{
						ReportRequestType:        v2.ComplianceReportStatus_ON_DEMAND,
						ReportNotificationMethod: v2.NotificationMethod_DOWNLOAD,
						StartedAt:                now,
						CompletedAt:              now,
						FailedClusters:           []*v2.FailedCluster{},
						RunState:                 v2.ComplianceReportStatus_GENERATED,
					}.Build(),
					ReportData: v2.ComplianceScanConfigurationStatus_builder{
						Id:       scanConfigID,
						ScanName: scanConfigID,
						ScanConfig: v2.BaseComplianceScanConfigurationSettings_builder{
							OneTimeScan: false,
							Profiles:    []string{},
							Notifiers:   []*v2.NotifierConfiguration{},
						}.Build(),
						ClusterStatus: []*v2.ClusterScanStatus{},
						ModifiedBy:    &v2.SlimUser{},
					}.Build(),
					User:                &v2.SlimUser{},
					IsDownloadAvailable: true,
				}.Build(),
			},
		}.Build(), res)
	})

	s.Run("Success with failed cluster and multiple errors", func() {
		scanConfigID := "scan-config-1"
		now := protocompat.TimestampNow()
		snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
			storage.ComplianceOperatorReportSnapshotV2_builder{
				ReportId:            "snapshot-1",
				ScanConfigurationId: scanConfigID,
				ReportStatus: storage.ComplianceOperatorReportStatus_builder{
					ReportRequestType:        storage.ComplianceOperatorReportStatus_SCHEDULED,
					ReportNotificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
					StartedAt:                now,
					CompletedAt:              now,
				}.Build(),
				FailedClusters: []*storage.ComplianceOperatorReportSnapshotV2_FailedCluster{
					storage.ComplianceOperatorReportSnapshotV2_FailedCluster_builder{
						ClusterId:       "cluster-1",
						ClusterName:     "cluster-1",
						OperatorVersion: "v1.6.0",
						Reasons:         []string{report.INTERNAL_ERROR, report.COMPLIANCE_VERSION_ERROR},
					}.Build(),
				},
			}.Build(),
		}
		sc := &storage.ComplianceOperatorScanConfigurationV2{}
		sc.SetId(scanConfigID)
		sc.SetScanConfigName(scanConfigID)

		s.snapshotDS.EXPECT().SearchSnapshots(allAccessContext, gomock.Any()).Return(snapshots, nil)
		s.scanConfigDatastore.EXPECT().GetScanConfiguration(allAccessContext, scanConfigID).Return(sc, true, nil)
		s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(allAccessContext, scanConfigID).Return(nil, nil)
		s.suiteDataStore.EXPECT().GetSuites(allAccessContext, gomock.Any()).Return(nil, nil)

		crhr := &v2.ComplianceReportHistoryRequest{}
		crhr.SetId(scanConfigID)
		res, err := s.service.GetReportHistory(allAccessContext, crhr)
		s.Require().NoError(err)
		protoassert.Equal(s.T(), v2.ComplianceReportHistoryResponse_builder{
			ComplianceReportSnapshots: []*v2.ComplianceReportSnapshot{
				v2.ComplianceReportSnapshot_builder{
					ReportJobId:  "snapshot-1",
					ScanConfigId: scanConfigID,
					ReportStatus: v2.ComplianceReportStatus_builder{
						ReportRequestType:        v2.ComplianceReportStatus_SCHEDULED,
						ReportNotificationMethod: v2.NotificationMethod_EMAIL,
						StartedAt:                now,
						CompletedAt:              now,
						FailedClusters: []*v2.FailedCluster{
							v2.FailedCluster_builder{
								ClusterId:       "cluster-1",
								ClusterName:     "cluster-1",
								OperatorVersion: "v1.6.0",
								Reason:          failedClusterReasonsJoinFunc([]string{report.INTERNAL_ERROR, report.COMPLIANCE_VERSION_ERROR}),
							}.Build(),
						},
					}.Build(),
					ReportData: v2.ComplianceScanConfigurationStatus_builder{
						Id:       scanConfigID,
						ScanName: scanConfigID,
						ScanConfig: v2.BaseComplianceScanConfigurationSettings_builder{
							OneTimeScan: false,
							Profiles:    []string{},
							Notifiers:   []*v2.NotifierConfiguration{},
						}.Build(),
						ClusterStatus: []*v2.ClusterScanStatus{},
						ModifiedBy:    &v2.SlimUser{},
					}.Build(),
					User:                &v2.SlimUser{},
					IsDownloadAvailable: false,
				}.Build(),
			},
		}.Build(), res)
	})

}

func (s *ComplianceScanConfigServiceTestSuite) TestGetMyReportHistory() {
	s.T().Setenv(features.ComplianceReporting.EnvVar(), "true")
	s.T().Setenv(features.ScanScheduleReportJobs.EnvVar(), "true")
	if !features.ComplianceReporting.Enabled() || !features.ScanScheduleReportJobs.Enabled() {
		s.T().Skipf("compliance reporting feature flag is disabled")
		s.T().SkipNow()
	}

	allAccessContext := sac.WithAllAccess(context.Background())

	s.Run("Invalid ID", func() {
		invalidID := ""
		crhr := &v2.ComplianceReportHistoryRequest{}
		crhr.SetId(invalidID)
		_, err := s.service.GetMyReportHistory(allAccessContext, crhr)
		s.Require().Error(err)
	})

	s.Run("Request Context does not have a User", func() {
		scanConfigID := "scan-config-1"

		crhr := &v2.ComplianceReportHistoryRequest{}
		crhr.SetId(scanConfigID)
		_, err := s.service.GetMyReportHistory(allAccessContext, crhr)
		s.Require().Error(err)
	})

	s.Run("Snapshot search error", func() {
		scanConfigID := "scan-config-1"
		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)

		s.snapshotDS.EXPECT().SearchSnapshots(ctx, gomock.Any()).Return(nil, errors.New("some error"))

		crhr := &v2.ComplianceReportHistoryRequest{}
		crhr.SetId(scanConfigID)
		_, err := s.service.GetMyReportHistory(ctx, crhr)
		s.Require().Error(err)
	})

	s.Run("Snapshot search found zero snapshots", func() {
		scanConfigID := "scan-config-1"
		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)

		s.snapshotDS.EXPECT().SearchSnapshots(ctx, gomock.Any()).Return(nil, nil)

		crhr := &v2.ComplianceReportHistoryRequest{}
		crhr.SetId(scanConfigID)
		res, err := s.service.GetMyReportHistory(ctx, crhr)
		s.Require().NoError(err)
		s.Require().Len(res.GetComplianceReportSnapshots(), 0)
	})

	s.Run("Snapshot search found nil snapshots", func() {
		scanConfigID := "scan-config-1"
		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)

		s.snapshotDS.EXPECT().SearchSnapshots(ctx, gomock.Any()).Return([]*storage.ComplianceOperatorReportSnapshotV2{nil, nil}, nil)

		crhr := &v2.ComplianceReportHistoryRequest{}
		crhr.SetId(scanConfigID)
		res, err := s.service.GetMyReportHistory(ctx, crhr)
		s.Require().NoError(err)
		s.Require().Len(res.GetComplianceReportSnapshots(), 0)
	})

	s.Run("Success", func() {
		scanConfigID := "scan-config-1"
		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		now := protocompat.TimestampNow()
		// Search succeed
		cors := &storage.ComplianceOperatorReportStatus{}
		cors.SetReportRequestType(storage.ComplianceOperatorReportStatus_SCHEDULED)
		cors.SetReportNotificationMethod(storage.ComplianceOperatorReportStatus_EMAIL)
		cors.SetStartedAt(now)
		cors.SetCompletedAt(now)
		corsv2 := &storage.ComplianceOperatorReportSnapshotV2{}
		corsv2.SetReportId("snapshot-1")
		corsv2.SetScanConfigurationId(scanConfigID)
		corsv2.SetReportStatus(cors)
		corsv2.SetUser(storageRequester)
		snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
			corsv2,
		}
		sc := &storage.ComplianceOperatorScanConfigurationV2{}
		sc.SetId(scanConfigID)
		sc.SetScanConfigName(scanConfigID)

		s.snapshotDS.EXPECT().SearchSnapshots(ctx, gomock.Any()).Return(snapshots, nil)
		s.scanConfigDatastore.EXPECT().GetScanConfiguration(ctx, scanConfigID).Return(sc, true, nil)
		s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(ctx, scanConfigID).Return(nil, nil)
		s.suiteDataStore.EXPECT().GetSuites(ctx, gomock.Any()).Return(nil, nil)

		crhr := &v2.ComplianceReportHistoryRequest{}
		crhr.SetId(scanConfigID)
		res, err := s.service.GetMyReportHistory(ctx, crhr)
		s.Require().NoError(err)
		protoassert.Equal(s.T(), v2.ComplianceReportHistoryResponse_builder{
			ComplianceReportSnapshots: []*v2.ComplianceReportSnapshot{
				v2.ComplianceReportSnapshot_builder{
					ReportJobId:  "snapshot-1",
					ScanConfigId: scanConfigID,
					ReportStatus: v2.ComplianceReportStatus_builder{
						ReportRequestType:        v2.ComplianceReportStatus_SCHEDULED,
						ReportNotificationMethod: v2.NotificationMethod_EMAIL,
						StartedAt:                now,
						CompletedAt:              now,
					}.Build(),
					ReportData: v2.ComplianceScanConfigurationStatus_builder{
						Id:       scanConfigID,
						ScanName: scanConfigID,
						ScanConfig: v2.BaseComplianceScanConfigurationSettings_builder{
							OneTimeScan: false,
							Profiles:    []string{},
							Notifiers:   []*v2.NotifierConfiguration{},
						}.Build(),
						ClusterStatus: []*v2.ClusterScanStatus{},
						ModifiedBy:    &v2.SlimUser{},
					}.Build(),
					User:                apiRequester,
					IsDownloadAvailable: false,
				}.Build(),
			},
		}.Build(), res)
	})
}

func (s *ComplianceScanConfigServiceTestSuite) TestDeleteReport() {
	s.T().Setenv(features.ComplianceReporting.EnvVar(), "true")
	s.T().Setenv(features.ScanScheduleReportJobs.EnvVar(), "true")
	if !features.ComplianceReporting.Enabled() || !features.ScanScheduleReportJobs.Enabled() {
		s.T().Skipf("compliance reporting feature flag is disabled")
		s.T().SkipNow()
	}

	allAccessContext := sac.WithAllAccess(context.Background())

	s.Run("Invalid ID", func() {
		invalidID := ""
		rbid := &v2.ResourceByID{}
		rbid.SetId(invalidID)
		_, err := s.service.DeleteReport(allAccessContext, rbid)
		s.Require().Error(err)
	})

	s.Run("User not present in context", func() {
		snapshotID := "snapshot-id"
		rbid := &v2.ResourceByID{}
		rbid.SetId(snapshotID)
		_, err := s.service.DeleteReport(allAccessContext, rbid)
		s.Require().Error(err)
	})

	s.Run("Snapshot Store error", func() {
		snapshotID := "snapshot-1"

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(nil, false, errors.New("some error"))

		rbid := &v2.ResourceByID{}
		rbid.SetId(snapshotID)
		_, err := s.service.DeleteReport(ctx, rbid)
		s.Require().Error(err)
	})

	s.Run("Snapshot not found", func() {
		snapshotID := "snapshot-1"

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(nil, false, nil)

		rbid := &v2.ResourceByID{}
		rbid.SetId(snapshotID)
		_, err := s.service.DeleteReport(ctx, rbid)
		s.Require().Error(err)
	})

	s.Run("Snapshot User differs from the User in the context", func() {
		snapshotID := "snapshot-id"
		slimUser := &storage.SlimUser{}
		slimUser.SetId("user-2")
		slimUser.SetName("user-2")
		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, slimUser)
		snapshot := getSnapshot(snapshotID, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		rbid := &v2.ResourceByID{}
		rbid.SetId(snapshotID)
		_, err := s.service.DeleteReport(ctx, rbid)
		s.Require().Error(err)
	})

	s.Run("Snapshot with notification method email", func() {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, storageRequester)
		snapshot.GetReportStatus().SetReportNotificationMethod(storage.ComplianceOperatorReportStatus_EMAIL)

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		rbid := &v2.ResourceByID{}
		rbid.SetId(snapshotID)
		_, err := s.service.DeleteReport(ctx, rbid)
		s.Require().Error(err)
	})

	s.Run("Snapshot with failure state", func() {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, storageRequester)
		snapshot.GetReportStatus().SetRunState(storage.ComplianceOperatorReportStatus_FAILURE)

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		rbid := &v2.ResourceByID{}
		rbid.SetId(snapshotID)
		_, err := s.service.DeleteReport(ctx, rbid)
		s.Require().Error(err)
	})

	s.Run("Snapshot with waiting state", func() {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, storageRequester)
		snapshot.GetReportStatus().SetRunState(storage.ComplianceOperatorReportStatus_WAITING)

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		rbid := &v2.ResourceByID{}
		rbid.SetId(snapshotID)
		_, err := s.service.DeleteReport(ctx, rbid)
		s.Require().Error(err)
	})

	s.Run("Snapshot with preparing state", func() {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, storageRequester)
		snapshot.GetReportStatus().SetRunState(storage.ComplianceOperatorReportStatus_PREPARING)

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		rbid := &v2.ResourceByID{}
		rbid.SetId(snapshotID)
		_, err := s.service.DeleteReport(ctx, rbid)
		s.Require().Error(err)
	})

	s.Run("Blob Store error", func() {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, storageRequester)

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)
		s.blobDS.EXPECT().Delete(gomock.Cond[context.Context](func(ctx context.Context) bool {
			return validateBlobContext(ctx, storage.Access_READ_WRITE_ACCESS)
		}), common.GetComplianceReportBlobPath(snapshot.GetScanConfigurationId(), snapshotID)).Return(errors.New("some error"))

		rbid := &v2.ResourceByID{}
		rbid.SetId(snapshotID)
		_, err := s.service.DeleteReport(ctx, rbid)
		s.Require().Error(err)
	})

	s.Run("Delete success", func() {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, storageRequester)

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)
		s.blobDS.EXPECT().Delete(gomock.Cond[context.Context](func(ctx context.Context) bool {
			return validateBlobContext(ctx, storage.Access_READ_WRITE_ACCESS)
		}), common.GetComplianceReportBlobPath(snapshot.GetScanConfigurationId(), snapshotID)).Return(nil)

		rbid := &v2.ResourceByID{}
		rbid.SetId(snapshotID)
		_, err := s.service.DeleteReport(ctx, rbid)
		s.Require().NoError(err)
	})
}

func getTestAPIStatusRec(createdTime, lastUpdatedTime time.Time) *apiV2.ComplianceScanConfigurationStatus {
	return apiV2.ComplianceScanConfigurationStatus_builder{
		Id:       uuid.NewDummy().String(),
		ScanName: "test-scan",
		ScanConfig: apiV2.BaseComplianceScanConfigurationSettings_builder{
			OneTimeScan:  false,
			Profiles:     []string{"ocp4-cis"},
			ScanSchedule: defaultAPISchedule,
			Description:  "test-description",
			Notifiers:    []*v2.NotifierConfiguration{},
		}.Build(),
		ClusterStatus: []*apiV2.ClusterScanStatus{
			apiV2.ClusterScanStatus_builder{
				ClusterId:   fixtureconsts.Cluster1,
				ClusterName: mockClusterName,
				Errors:      []string{"This binding is not ready", "Error 1", "Error 2", "Error 3"},
				SuiteStatus: apiV2.ClusterScanStatus_SuiteStatus_builder{
					Phase:              "DONE",
					Result:             "NON-COMPLIANT",
					LastTransitionTime: protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
				}.Build(),
			}.Build(),
		},
		CreatedTime:      protoconv.ConvertTimeToTimestamp(createdTime),
		LastUpdatedTime:  protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
		ModifiedBy:       apiRequester,
		LastExecutedTime: protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
	}.Build()
}

func getTestAPIRec() *apiV2.ComplianceScanConfiguration {
	bcscs := &apiV2.BaseComplianceScanConfigurationSettings{}
	bcscs.SetOneTimeScan(false)
	bcscs.SetProfiles([]string{"ocp4-cis"})
	bcscs.SetScanSchedule(defaultAPISchedule)
	bcscs.SetDescription("test-description")
	csc := &apiV2.ComplianceScanConfiguration{}
	csc.SetScanName("test-scan")
	csc.SetScanConfig(bcscs)
	csc.SetClusters([]string{fixtureconsts.Cluster1})
	return csc
}

func getContextForUser(t *testing.T, ctrl *gomock.Controller, ctx context.Context, user *storage.SlimUser) context.Context {
	mockID := mockIdentity.NewMockIdentity(ctrl)
	mockID.EXPECT().UID().Return(user.GetId()).AnyTimes()
	mockID.EXPECT().FullName().Return(user.GetName()).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(user.GetName()).AnyTimes()
	return authn.ContextWithIdentity(ctx, mockID, t)
}

func validateBlobContext(ctx context.Context, access storage.Access) bool {
	scopeChecker := sac.ForResource(resources.Administration)
	return scopeChecker.ScopeChecker(ctx, access).IsAllowed()
}
