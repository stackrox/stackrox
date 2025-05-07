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
	defaultStorageSchedule = &storage.Schedule{
		IntervalType: 2,
		Hour:         15,
		Minute:       0,
		Interval: &storage.Schedule_DaysOfWeek_{
			DaysOfWeek: &storage.Schedule_DaysOfWeek{
				Days: []int32{1, 2, 3, 4, 5, 6, 7},
			},
		},
	}

	defaultAPISchedule = &apiV2.Schedule{
		IntervalType: 1,
		Hour:         15,
		Minute:       0,
		Interval: &apiV2.Schedule_DaysOfWeek_{
			DaysOfWeek: &apiV2.Schedule_DaysOfWeek{
				Days: []int32{1, 2, 3, 4, 5, 6, 7},
			},
		},
	}

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
	request.ScanName = "test@scan"
	request.Id = uuid.NewDummy().String()
	processResponse := convertV2ScanConfigToStorage(allAccessContext, request)
	processResponse.Id = uuid.NewDummy().String()

	_, err := s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)

	request.ScanName = "testscan_"
	_, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)

	request.ScanName = "default"
	_, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Contains(err.Error(), "Scan configuration name \"default\" cannot be used as it is reserved by the Compliance Operator")

	request.ScanName = "default-auto-apply"
	_, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Contains(err.Error(), "Scan configuration name \"default-auto-apply\" cannot be used as it is reserved by the Compliance Operator")
}

func (s *ComplianceScanConfigServiceTestSuite) TestCreateComplianceScanConfiguration() {
	allAccessContext := sac.WithAllAccess(context.Background())

	request := getTestAPIRec()
	request.Id = uuid.NewDummy().String()
	storageRequest := convertV2ScanConfigToStorage(allAccessContext, request)
	processResponse := convertV2ScanConfigToStorage(allAccessContext, request)
	processResponse.Id = uuid.NewDummy().String()
	s.manager.EXPECT().ProcessScanRequest(gomock.Any(), storageRequest, []string{fixtureconsts.Cluster1}).Return(processResponse, nil).Times(1)
	s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(allAccessContext, uuid.NewDummy().String()).Return([]*storage.ComplianceOperatorClusterScanConfigStatus{
		{
			ClusterId:    fixtureconsts.Cluster1,
			ScanConfigId: uuid.NewDummy().String(),
			Errors:       []string{"Error 1", "Error 2", "Error 3"},
		},
	}, nil).Times(1)

	config, err := s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().NoError(err)
	// ID will be added to the record and returned.  Add it to the validation object
	request.Id = uuid.NewDummy().String()
	protoassert.Equal(s.T(), request, config)

	// reset for error testing
	request = getTestAPIRec()
	request.ScanConfig = nil
	config, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "The scan configuration is nil.")
	s.Require().Nil(config)

	request = getTestAPIRec()
	request.Clusters = []string{}
	config, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "At least one cluster is required for a scan configuration")
	s.Require().Nil(config)

	request = getTestAPIRec()
	request.ScanConfig.Profiles = []string{}
	config, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "At least one profile is required for a scan configuration")
	s.Require().Nil(config)
}

func (s *ComplianceScanConfigServiceTestSuite) TestUpdateComplianceScanConfiguration() {
	allAccessContext := sac.WithAllAccess(context.Background())

	request := getTestAPIRec()
	request.Id = uuid.NewDummy().String()
	storageRequest := convertV2ScanConfigToStorage(allAccessContext, request)
	processResponse := convertV2ScanConfigToStorage(allAccessContext, request)
	processResponse.Id = uuid.NewDummy().String()
	s.manager.EXPECT().UpdateScanRequest(gomock.Any(), storageRequest, []string{fixtureconsts.Cluster1}).Return(processResponse, nil).Times(1)

	_, err := s.service.UpdateComplianceScanConfiguration(allAccessContext, request)
	s.Require().NoError(err)

	// Test Case 2: Update with Empty ID
	request.Id = ""
	_, err = s.service.UpdateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "Scan configuration ID is required: invalid arguments")

	// Test Case 3: No ScanConfig
	request = getTestAPIRec()
	request.Id = uuid.NewDummy().String()
	request.ScanConfig = nil
	_, err = s.service.UpdateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "The scan configuration is nil.")

	// Test Case 4: No clusters
	request = getTestAPIRec()
	request.Id = uuid.NewDummy().String()
	request.Clusters = []string{}
	_, err = s.service.UpdateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "At least one cluster is required for a scan configuration")

	// Test Case 5: No profiles
	request = getTestAPIRec()
	request.Id = uuid.NewDummy().String()
	request.ScanConfig.Profiles = []string{}
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
	s.blobDS.EXPECT().Delete(gomock.Any(), common.GetComplianceReportBlobPath(validID, snapshotID)).Times(1).Return(nil)

	_, err := s.service.DeleteComplianceScanConfiguration(allAccessContext, &v2.ResourceByID{Id: validID})
	s.Require().NoError(err)

	// Test Case 2: Deletion with Empty ID
	_, err = s.service.DeleteComplianceScanConfiguration(allAccessContext, &v2.ResourceByID{Id: ""})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "Scan configuration ID is required for deletion")

	// Test Case 3: Deletion Fails in Manager
	failingID := "failingScanConfigID"
	s.manager.EXPECT().DeleteScan(gomock.Any(), failingID).Return(errors.New("manager error")).Times(1)
	s.snapshotDS.EXPECT().SearchSnapshots(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)

	_, err = s.service.DeleteComplianceScanConfiguration(allAccessContext, &v2.ResourceByID{Id: failingID})
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
			query:          &apiV2.RawQuery{Query: ""},
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
		},
		{
			desc:  "Query with search field",
			query: &apiV2.RawQuery{Query: "Cluster ID:id"},
			expectedQ: search.NewQueryBuilder().AddStrings(search.ClusterID, "id").
				WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.NewQueryBuilder().AddStrings(search.ClusterID, "id").ProtoQuery(),
		},
		{
			desc: "Query with custom pagination",
			query: &apiV2.RawQuery{
				Query:      "",
				Pagination: &apiV2.Pagination{Limit: 1},
			},
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(1)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			expectedResp := &apiV2.ListComplianceScanConfigurationsResponse{
				Configurations: []*apiV2.ComplianceScanConfigurationStatus{
					getTestAPIStatusRec(createdTime, lastUpdatedTime),
				},
				TotalCount: 6,
			}

			s.scanConfigDatastore.EXPECT().GetScanConfigurations(allAccessContext, tc.expectedQ).
				Return([]*storage.ComplianceOperatorScanConfigurationV2{
					{
						Id:                     uuid.NewDummy().String(),
						ScanConfigName:         "test-scan",
						AutoApplyRemediations:  false,
						AutoUpdateRemediations: false,
						OneTimeScan:            false,
						Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
							{
								ProfileName: "ocp4-cis",
							},
						},
						StrictNodeScan:  false,
						Schedule:        defaultStorageSchedule,
						CreatedTime:     protoconv.ConvertTimeToTimestamp(createdTime),
						LastUpdatedTime: protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
						ModifiedBy:      storageRequester,
						Description:     "test-description",
						Notifiers:       []*storage.NotifierConfiguration{},
					},
				}, nil).Times(1)

			s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(allAccessContext, uuid.NewDummy().String()).Return([]*storage.ComplianceOperatorClusterScanConfigStatus{
				{
					ClusterId:    fixtureconsts.Cluster1,
					ClusterName:  mockClusterName,
					ScanConfigId: uuid.NewDummy().String(),
					Errors:       []string{"Error 1", "Error 2", "Error 3"},
				},
			}, nil).Times(1)

			s.scanSettingBindingDatastore.EXPECT().GetScanSettingBindings(allAccessContext, gomock.Any()).Return([]*storage.ComplianceOperatorScanSettingBindingV2{
				{
					ClusterId: fixtureconsts.Cluster1,
					Status: &storage.ComplianceOperatorStatus{
						Phase: "READY",
						Conditions: []*storage.ComplianceOperatorCondition{
							{
								Type:    "READY",
								Status:  "False",
								Message: "This binding is not ready",
							},
						},
					},
				},
			}, nil).Times(1)
			s.suiteDataStore.EXPECT().GetSuites(allAccessContext, gomock.Any()).Return([]*storage.ComplianceOperatorSuiteV2{
				{
					Id:        uuid.NewDummy().String(),
					ClusterId: fixtureconsts.Cluster1,
					Status: &storage.ComplianceOperatorStatus{
						Phase:  "DONE",
						Result: "NON-COMPLIANT",
						Conditions: []*storage.ComplianceOperatorCondition{
							{
								Type:               "Processing",
								Status:             "False",
								LastTransitionTime: protocompat.GetProtoTimestampFromSeconds(lastUpdatedTime.UTC().Unix() - 10),
							},
							{
								Type:               "Ready",
								Status:             "True",
								LastTransitionTime: protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
							},
						},
					},
				},
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
				s.scanConfigDatastore.EXPECT().GetScanConfiguration(allAccessContext, tc.scanID).
					Return(&storage.ComplianceOperatorScanConfigurationV2{
						Id:                     uuid.NewDummy().String(),
						ScanConfigName:         "test-scan",
						AutoApplyRemediations:  false,
						AutoUpdateRemediations: false,
						OneTimeScan:            false,
						Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
							{
								ProfileName: "ocp4-cis",
							},
						},
						StrictNodeScan:  false,
						Schedule:        defaultStorageSchedule,
						CreatedTime:     protoconv.ConvertTimeToTimestamp(createdTime),
						LastUpdatedTime: protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
						ModifiedBy:      storageRequester,
						Description:     "test-description",
						Notifiers:       []*storage.NotifierConfiguration{},
					}, true, nil).Times(1)

				s.suiteDataStore.EXPECT().GetSuites(allAccessContext, gomock.Any()).Return([]*storage.ComplianceOperatorSuiteV2{
					{
						Id:        uuid.NewDummy().String(),
						ClusterId: fixtureconsts.Cluster1,
						Name:      "test-scan",
						Status: &storage.ComplianceOperatorStatus{
							Phase:  "DONE",
							Result: "NON-COMPLIANT",
							Conditions: []*storage.ComplianceOperatorCondition{
								{
									Type:               "Ready",
									Status:             "True",
									LastTransitionTime: protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
								},
								{
									Type:               "Processing",
									Status:             "False",
									LastTransitionTime: protocompat.GetProtoTimestampFromSeconds(lastUpdatedTime.UTC().Unix() - 10),
								},
							},
						},
					},
				}, nil).Times(1)

				s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(allAccessContext, uuid.NewDummy().String()).Return([]*storage.ComplianceOperatorClusterScanConfigStatus{
					{
						ClusterId:    fixtureconsts.Cluster1,
						ClusterName:  mockClusterName,
						ScanConfigId: uuid.NewDummy().String(),
						Errors:       []string{"Error 1", "Error 2", "Error 3"},
					},
				}, nil).Times(1)

				s.scanSettingBindingDatastore.EXPECT().GetScanSettingBindings(allAccessContext, gomock.Any()).Return([]*storage.ComplianceOperatorScanSettingBindingV2{
					{
						ClusterId: fixtureconsts.Cluster1,
						Status: &storage.ComplianceOperatorStatus{
							Phase: "READY",
							Conditions: []*storage.ComplianceOperatorCondition{
								{
									Type:    "READY",
									Status:  "False",
									Message: "This binding is not ready",
								},
							},
						},
					},
				}, nil).Times(1)
			} else {
				s.scanConfigDatastore.EXPECT().GetScanConfiguration(allAccessContext, tc.scanID).
					Return(nil, false, errors.New("record not found")).Times(1)
			}

			config, err := s.service.GetComplianceScanConfiguration(allAccessContext, &apiV2.ResourceByID{Id: tc.scanID})
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
			query:          &apiV2.RawQuery{Query: ""},
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
		},
		{
			desc:  "Query with search field",
			query: &apiV2.RawQuery{Query: "Cluster ID:id"},
			expectedQ: search.NewQueryBuilder().AddStrings(search.ClusterID, "id").
				WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			expectedCountQ: search.NewQueryBuilder().AddStrings(search.ClusterID, "id").ProtoQuery(),
		},
		{
			desc: "Query with custom pagination",
			query: &apiV2.RawQuery{
				Query:      "",
				Pagination: &apiV2.Pagination{Limit: 1},
			},
			expectedQ:      search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(1)).ProtoQuery(),
			expectedCountQ: search.EmptyQuery(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			expectedResp := &apiV2.ListComplianceScanConfigsProfileResponse{
				Profiles:   nil,
				TotalCount: 6,
			}

			s.scanConfigDatastore.EXPECT().GetProfilesNames(gomock.Any(), tc.query).Return([]string{"ocp4"}, nil).Times(1)
			s.scanConfigDatastore.EXPECT().CountDistinctProfiles(gomock.Any(), tc.expectedCountQ).Return(1, nil).Times(1)

			searchQuery := search.NewQueryBuilder().AddSelectFields().AddExactMatches(search.ComplianceOperatorProfileName, "ocp4").ProtoQuery()
			searchQuery.Pagination = &v1.QueryPagination{}

			profiles := []*storage.ComplianceOperatorProfileV2{
				{
					Name:           "ocp4",
					ProductType:    "platform",
					Description:    "this is a test",
					Title:          "A Title",
					ProfileVersion: "version 1",
					Rules: []*storage.ComplianceOperatorProfileV2_Rule{
						{
							RuleName: "test 1",
						},
						{
							RuleName: "test 2",
						},
						{
							RuleName: "test 3",
						},
						{
							RuleName: "test 4",
						},
						{
							RuleName: "test 5",
						},
					},
				},
			}
			s.profileDS.EXPECT().SearchProfiles(gomock.Any(), searchQuery).Return(profiles, nil).Times(1)

			for _, profile := range profiles {
				s.benchmarkDS.EXPECT().GetBenchmarksByProfileName(s.ctx, profile.GetName()).Return([]*storage.ComplianceOperatorBenchmarkV2{{
					Id:        uuid.NewV4().String(),
					Name:      "CIS",
					ShortName: "OCP_CIS",
					Version:   "1-5",
				}}, nil).Times(1)
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

	_, err := s.service.RunComplianceScanConfiguration(allAccessContext, &v2.ResourceByID{Id: validID})
	s.Require().NoError(err)
}

func (s *ComplianceScanConfigServiceTestSuite) TestRunComplianceScanConfigurationWithInvalidScanConfigIdFails() {
	allAccessContext := sac.WithAllAccess(context.Background())

	invalidID := ""
	_, err := s.service.RunComplianceScanConfiguration(allAccessContext, &v2.ResourceByID{Id: invalidID})
	s.Require().Error(err)
}

func (s *ComplianceScanConfigServiceTestSuite) TestRunReport() {
	s.T().Setenv(features.ComplianceReporting.EnvVar(), "true")
	if !features.ComplianceReporting.Enabled() {
		s.T().Skip("Skip test when compliance reporting feature flag is disabled")
		s.T().SkipNow()
	}

	allAccessContext := sac.WithAllAccess(context.Background())

	user := &storage.SlimUser{
		Id:   "user-1",
		Name: "user-1",
	}

	ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, user)

	invalidID := ""
	_, err := s.service.RunReport(ctx, &v2.ComplianceRunReportRequest{ScanConfigId: invalidID})
	s.Require().Error(err)

	nonExistentScanConfigID := "does-not-exist-scan-config-1"
	s.scanConfigDatastore.EXPECT().GetScanConfiguration(ctx, nonExistentScanConfigID).Return(nil, false, nil)
	_, err = s.service.RunReport(ctx, &v2.ComplianceRunReportRequest{ScanConfigId: nonExistentScanConfigID})
	s.Require().Error(err)

	validScanConfigID := "scan-config-1"
	validScanConfig := &storage.ComplianceOperatorScanConfigurationV2{
		Id:             "scan-config-1",
		ScanConfigName: "scan-config-1",
	}
	s.scanConfigDatastore.EXPECT().GetScanConfiguration(ctx, validScanConfigID).Return(validScanConfig, true, nil)
	s.reportManager.EXPECT().SubmitReportRequest(ctx, validScanConfig, storage.ComplianceOperatorReportStatus_EMAIL).Return(nil)

	resp, err := s.service.RunReport(ctx, &v2.ComplianceRunReportRequest{
		ScanConfigId:             validScanConfigID,
		ReportNotificationMethod: v2.NotificationMethod_EMAIL,
	})
	s.Require().NoError(err)
	s.Equal(v2.ComplianceRunReportResponse_SUBMITTED, resp.RunState, "Failed to submit report")

	s.scanConfigDatastore.EXPECT().GetScanConfiguration(ctx, validScanConfigID).Return(validScanConfig, true, nil)
	s.reportManager.EXPECT().SubmitReportRequest(ctx, validScanConfig, storage.ComplianceOperatorReportStatus_DOWNLOAD).Return(nil)

	resp, err = s.service.RunReport(ctx, &v2.ComplianceRunReportRequest{
		ScanConfigId:             validScanConfigID,
		ReportNotificationMethod: v2.NotificationMethod_DOWNLOAD,
	})
	s.Require().NoError(err)
	s.Equal(v2.ComplianceRunReportResponse_SUBMITTED, resp.RunState, "Failed to submit report")
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
		_, err := s.service.GetReportHistory(allAccessContext, &v2.ComplianceReportHistoryRequest{Id: invalidID})
		s.Require().Error(err)
	})

	s.Run("Snapshot search error", func() {
		scanConfigID := "scan-config-1"

		s.snapshotDS.EXPECT().SearchSnapshots(allAccessContext, gomock.Any()).Return(nil, errors.New("some error"))

		_, err := s.service.GetReportHistory(allAccessContext, &v2.ComplianceReportHistoryRequest{Id: scanConfigID})
		s.Require().Error(err)
	})

	s.Run("Success", func() {
		scanConfigID := "scan-config-1"
		now := protocompat.TimestampNow()
		snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
			{
				ReportId:            "snapshot-1",
				ScanConfigurationId: scanConfigID,
				ReportStatus: &storage.ComplianceOperatorReportStatus{
					ReportRequestType:        storage.ComplianceOperatorReportStatus_SCHEDULED,
					ReportNotificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
					StartedAt:                now,
					CompletedAt:              now,
				},
				FailedClusters: []*storage.ComplianceOperatorReportSnapshotV2_FailedCluster{
					{
						ClusterId:       "cluster-1",
						ClusterName:     "cluster-1",
						OperatorVersion: "v1.6.0",
						Reasons:         []string{report.INTERNAL_ERROR},
					},
				},
			},
		}
		sc := &storage.ComplianceOperatorScanConfigurationV2{
			Id:             scanConfigID,
			ScanConfigName: scanConfigID,
		}

		s.snapshotDS.EXPECT().SearchSnapshots(allAccessContext, gomock.Any()).Return(snapshots, nil)
		s.scanConfigDatastore.EXPECT().GetScanConfiguration(allAccessContext, scanConfigID).Return(sc, true, nil)
		s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(allAccessContext, scanConfigID).Return(nil, nil)
		s.suiteDataStore.EXPECT().GetSuites(allAccessContext, gomock.Any()).Return(nil, nil)

		res, err := s.service.GetReportHistory(allAccessContext, &v2.ComplianceReportHistoryRequest{Id: scanConfigID})
		s.Require().NoError(err)
		protoassert.Equal(s.T(), &v2.ComplianceReportHistoryResponse{
			ComplianceReportSnapshots: []*v2.ComplianceReportSnapshot{
				{
					ReportJobId:  "snapshot-1",
					ScanConfigId: scanConfigID,
					ReportStatus: &v2.ComplianceReportStatus{
						ReportRequestType:        v2.ComplianceReportStatus_SCHEDULED,
						ReportNotificationMethod: v2.NotificationMethod_EMAIL,
						StartedAt:                now,
						CompletedAt:              now,
						FailedClusters: []*v2.FailedCluster{
							{
								ClusterId:       "cluster-1",
								ClusterName:     "cluster-1",
								OperatorVersion: "v1.6.0",
								Reason:          report.INTERNAL_ERROR,
							},
						},
					},
					ReportData: &v2.ComplianceScanConfigurationStatus{
						Id:       scanConfigID,
						ScanName: scanConfigID,
						ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
							OneTimeScan: false,
							Profiles:    []string{},
							Notifiers:   []*v2.NotifierConfiguration{},
						},
						ClusterStatus: []*v2.ClusterScanStatus{},
						ModifiedBy:    &v2.SlimUser{},
					},
					User:                &v2.SlimUser{},
					IsDownloadAvailable: false,
				},
			},
		}, res)
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
		_, err := s.service.GetMyReportHistory(allAccessContext, &v2.ComplianceReportHistoryRequest{Id: invalidID})
		s.Require().Error(err)
	})

	s.Run("Request Context does not have a User", func() {
		scanConfigID := "scan-config-1"

		_, err := s.service.GetMyReportHistory(allAccessContext, &v2.ComplianceReportHistoryRequest{Id: scanConfigID})
		s.Require().Error(err)
	})

	s.Run("Snapshot search error", func() {
		scanConfigID := "scan-config-1"
		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)

		s.snapshotDS.EXPECT().SearchSnapshots(ctx, gomock.Any()).Return(nil, errors.New("some error"))

		_, err := s.service.GetMyReportHistory(ctx, &v2.ComplianceReportHistoryRequest{Id: scanConfigID})
		s.Require().Error(err)
	})

	s.Run("Snapshot search found zero snapshots", func() {
		scanConfigID := "scan-config-1"
		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)

		s.snapshotDS.EXPECT().SearchSnapshots(ctx, gomock.Any()).Return(nil, nil)

		res, err := s.service.GetMyReportHistory(ctx, &v2.ComplianceReportHistoryRequest{Id: scanConfigID})
		s.Require().NoError(err)
		s.Require().Len(res.GetComplianceReportSnapshots(), 0)
	})

	s.Run("Snapshot search found nil snapshots", func() {
		scanConfigID := "scan-config-1"
		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)

		s.snapshotDS.EXPECT().SearchSnapshots(ctx, gomock.Any()).Return([]*storage.ComplianceOperatorReportSnapshotV2{nil, nil}, nil)

		res, err := s.service.GetMyReportHistory(ctx, &v2.ComplianceReportHistoryRequest{Id: scanConfigID})
		s.Require().NoError(err)
		s.Require().Len(res.GetComplianceReportSnapshots(), 0)
	})

	s.Run("Success", func() {
		scanConfigID := "scan-config-1"
		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		now := protocompat.TimestampNow()
		// Search succeed
		snapshots := []*storage.ComplianceOperatorReportSnapshotV2{
			{
				ReportId:            "snapshot-1",
				ScanConfigurationId: scanConfigID,
				ReportStatus: &storage.ComplianceOperatorReportStatus{
					ReportRequestType:        storage.ComplianceOperatorReportStatus_SCHEDULED,
					ReportNotificationMethod: storage.ComplianceOperatorReportStatus_EMAIL,
					StartedAt:                now,
					CompletedAt:              now,
				},
				User: storageRequester,
			},
		}
		sc := &storage.ComplianceOperatorScanConfigurationV2{
			Id:             scanConfigID,
			ScanConfigName: scanConfigID,
		}

		s.snapshotDS.EXPECT().SearchSnapshots(ctx, gomock.Any()).Return(snapshots, nil)
		s.scanConfigDatastore.EXPECT().GetScanConfiguration(ctx, scanConfigID).Return(sc, true, nil)
		s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(ctx, scanConfigID).Return(nil, nil)
		s.suiteDataStore.EXPECT().GetSuites(ctx, gomock.Any()).Return(nil, nil)

		res, err := s.service.GetMyReportHistory(ctx, &v2.ComplianceReportHistoryRequest{Id: scanConfigID})
		s.Require().NoError(err)
		protoassert.Equal(s.T(), &v2.ComplianceReportHistoryResponse{
			ComplianceReportSnapshots: []*v2.ComplianceReportSnapshot{
				{
					ReportJobId:  "snapshot-1",
					ScanConfigId: scanConfigID,
					ReportStatus: &v2.ComplianceReportStatus{
						ReportRequestType:        v2.ComplianceReportStatus_SCHEDULED,
						ReportNotificationMethod: v2.NotificationMethod_EMAIL,
						StartedAt:                now,
						CompletedAt:              now,
					},
					ReportData: &v2.ComplianceScanConfigurationStatus{
						Id:       scanConfigID,
						ScanName: scanConfigID,
						ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
							OneTimeScan: false,
							Profiles:    []string{},
							Notifiers:   []*v2.NotifierConfiguration{},
						},
						ClusterStatus: []*v2.ClusterScanStatus{},
						ModifiedBy:    &v2.SlimUser{},
					},
					User:                apiRequester,
					IsDownloadAvailable: false,
				},
			},
		}, res)
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
		_, err := s.service.DeleteReport(allAccessContext, &v2.ResourceByID{Id: invalidID})
		s.Require().Error(err)
	})

	s.Run("User not present in context", func() {
		snapshotID := "snapshot-id"
		_, err := s.service.DeleteReport(allAccessContext, &v2.ResourceByID{Id: snapshotID})
		s.Require().Error(err)
	})

	s.Run("Snapshot Store error", func() {
		snapshotID := "snapshot-1"

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(nil, false, errors.New("some error"))

		_, err := s.service.DeleteReport(ctx, &v2.ResourceByID{Id: snapshotID})
		s.Require().Error(err)
	})

	s.Run("Snapshot not found", func() {
		snapshotID := "snapshot-1"

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(nil, false, nil)

		_, err := s.service.DeleteReport(ctx, &v2.ResourceByID{Id: snapshotID})
		s.Require().Error(err)
	})

	s.Run("Snapshot User differs from the User in the context", func() {
		snapshotID := "snapshot-id"
		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, &storage.SlimUser{
			Id:   "user-2",
			Name: "user-2",
		})
		snapshot := getSnapshot(snapshotID, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		_, err := s.service.DeleteReport(ctx, &v2.ResourceByID{Id: snapshotID})
		s.Require().Error(err)
	})

	s.Run("Snapshot with notification method email", func() {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, storageRequester)
		snapshot.GetReportStatus().ReportNotificationMethod = storage.ComplianceOperatorReportStatus_EMAIL

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		_, err := s.service.DeleteReport(ctx, &v2.ResourceByID{Id: snapshotID})
		s.Require().Error(err)
	})

	s.Run("Snapshot with failure state", func() {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, storageRequester)
		snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_FAILURE

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		_, err := s.service.DeleteReport(ctx, &v2.ResourceByID{Id: snapshotID})
		s.Require().Error(err)
	})

	s.Run("Snapshot with waiting state", func() {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, storageRequester)
		snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_WAITING

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		_, err := s.service.DeleteReport(ctx, &v2.ResourceByID{Id: snapshotID})
		s.Require().Error(err)
	})

	s.Run("Snapshot with preparing state", func() {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, storageRequester)
		snapshot.GetReportStatus().RunState = storage.ComplianceOperatorReportStatus_PREPARING

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)

		_, err := s.service.DeleteReport(ctx, &v2.ResourceByID{Id: snapshotID})
		s.Require().Error(err)
	})

	s.Run("Blob Store error", func() {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, storageRequester)

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)
		s.blobDS.EXPECT().Delete(gomock.Any(), common.GetComplianceReportBlobPath(snapshot.GetScanConfigurationId(), snapshotID)).Return(errors.New("some error"))

		_, err := s.service.DeleteReport(ctx, &v2.ResourceByID{Id: snapshotID})
		s.Require().Error(err)
	})

	s.Run("Delete success", func() {
		snapshotID := "snapshot-1"
		snapshot := getSnapshot(snapshotID, storageRequester)

		ctx := getContextForUser(s.T(), s.mockCtrl, allAccessContext, storageRequester)
		s.snapshotDS.EXPECT().GetSnapshot(gomock.Any(), snapshotID).Return(snapshot, true, nil)
		s.blobDS.EXPECT().Delete(gomock.Any(), common.GetComplianceReportBlobPath(snapshot.GetScanConfigurationId(), snapshotID)).Return(nil)

		_, err := s.service.DeleteReport(ctx, &v2.ResourceByID{Id: snapshotID})
		s.Require().NoError(err)
	})
}

func getTestAPIStatusRec(createdTime, lastUpdatedTime time.Time) *apiV2.ComplianceScanConfigurationStatus {
	return &apiV2.ComplianceScanConfigurationStatus{
		Id:       uuid.NewDummy().String(),
		ScanName: "test-scan",
		ScanConfig: &apiV2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{"ocp4-cis"},
			ScanSchedule: defaultAPISchedule,
			Description:  "test-description",
			Notifiers:    []*v2.NotifierConfiguration{},
		},
		ClusterStatus: []*apiV2.ClusterScanStatus{
			{
				ClusterId:   fixtureconsts.Cluster1,
				ClusterName: mockClusterName,
				Errors:      []string{"This binding is not ready", "Error 1", "Error 2", "Error 3"},
				SuiteStatus: &apiV2.ClusterScanStatus_SuiteStatus{
					Phase:              "DONE",
					Result:             "NON-COMPLIANT",
					LastTransitionTime: protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
				},
			},
		},
		CreatedTime:      protoconv.ConvertTimeToTimestamp(createdTime),
		LastUpdatedTime:  protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
		ModifiedBy:       apiRequester,
		LastExecutedTime: protoconv.ConvertTimeToTimestamp(lastUpdatedTime),
	}
}

func getTestAPIRec() *apiV2.ComplianceScanConfiguration {
	return &apiV2.ComplianceScanConfiguration{
		ScanName: "test-scan",
		ScanConfig: &apiV2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{"ocp4-cis"},
			ScanSchedule: defaultAPISchedule,
			Description:  "test-description",
		},
		Clusters: []string{fixtureconsts.Cluster1},
	}
}

func getContextForUser(t *testing.T, ctrl *gomock.Controller, ctx context.Context, user *storage.SlimUser) context.Context {
	mockID := mockIdentity.NewMockIdentity(ctrl)
	mockID.EXPECT().UID().Return(user.Id).AnyTimes()
	mockID.EXPECT().FullName().Return(user.Name).AnyTimes()
	mockID.EXPECT().FriendlyName().Return(user.Name).AnyTimes()
	return authn.ContextWithIdentity(ctx, mockID, t)
}
