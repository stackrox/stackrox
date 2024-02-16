package service

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	managerMocks "github.com/stackrox/rox/central/complianceoperator/v2/compliancemanager/mocks"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/testutils"
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

	ctx                 context.Context
	manager             *managerMocks.MockManager
	scanConfigDatastore *scanConfigMocks.MockDataStore
	service             Service
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
	s.scanConfigDatastore = scanConfigMocks.NewMockDataStore(s.mockCtrl)
	s.service = New(s.scanConfigDatastore, s.manager)
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
	s.Require().Equal(request, config)

	// reset for error testing
	request = getTestAPIRec()
	request.ScanConfig = nil
	config, err = s.service.CreateComplianceScanConfiguration(allAccessContext, request)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "At least one profile is required for a scan configuration")
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
	s.Require().Contains(err.Error(), "At least one profile is required for a scan configuration")

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
	s.manager.EXPECT().DeleteScan(gomock.Any(), validID).Return(nil).Times(1)

	_, err := s.service.DeleteComplianceScanConfiguration(allAccessContext, &v2.ResourceByID{Id: validID})
	s.Require().NoError(err)

	// Test Case 2: Deletion with Empty ID
	_, err = s.service.DeleteComplianceScanConfiguration(allAccessContext, &v2.ResourceByID{Id: ""})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "Scan configuration ID is required for deletion")

	// Test Case 3: Deletion Fails in Manager
	failingID := "failingScanConfigID"
	s.manager.EXPECT().DeleteScan(gomock.Any(), failingID).Return(errors.New("manager error")).Times(1)

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
	createdTime := timestamp.Now().GogoProtobuf()
	lastUpdatedTime := timestamp.Now().GogoProtobuf()

	testCases := []struct {
		desc      string
		query     *apiV2.RawQuery
		expectedQ *v1.Query
	}{
		{
			desc:      "Empty query",
			query:     &apiV2.RawQuery{Query: ""},
			expectedQ: search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
		},
		{
			desc:  "Query with search field",
			query: &apiV2.RawQuery{Query: "Cluster ID:id"},
			expectedQ: search.NewQueryBuilder().AddStrings(search.ClusterID, "id").
				WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
		},
		{
			desc: "Query with custom pagination",
			query: &apiV2.RawQuery{
				Query:      "",
				Pagination: &apiV2.Pagination{Limit: 1},
			},
			expectedQ: search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(1)).ProtoQuery(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {
			expectedResp := &apiV2.ListComplianceScanConfigurationsResponse{
				Configurations: []*apiV2.ComplianceScanConfigurationStatus{
					getTestAPIStatusRec(createdTime, lastUpdatedTime),
				},
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
						CreatedTime:     createdTime,
						LastUpdatedTime: lastUpdatedTime,
						ModifiedBy:      storageRequester,
						Description:     "test-description",
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

			configs, err := s.service.ListComplianceScanConfigurations(allAccessContext, tc.query)
			s.Require().NoError(err)
			s.Require().Equal(expectedResp, configs)
		})
	}
}

func (s *ComplianceScanConfigServiceTestSuite) TestCountComplianceScanConfigurations() {
	allAccessContext := sac.WithAllAccess(context.Background())

	testCases := []struct {
		desc      string
		query     *apiV2.RawQuery
		expectedQ *v1.Query
	}{
		{
			desc:      "Empty query",
			query:     &apiV2.RawQuery{Query: ""},
			expectedQ: search.NewQueryBuilder().ProtoQuery(),
		},
		{
			desc:      "Query with search field",
			query:     &apiV2.RawQuery{Query: "Cluster ID:id"},
			expectedQ: search.NewQueryBuilder().AddStrings(search.ClusterID, "id").ProtoQuery(),
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.desc, func(t *testing.T) {

			s.scanConfigDatastore.EXPECT().CountScanConfigurations(allAccessContext, tc.expectedQ).
				Return(1, nil).Times(1)

			configs, err := s.service.GetComplianceScanConfigurationsCount(allAccessContext, tc.query)
			s.Require().NoError(err)
			s.Require().Equal(int32(1), configs.Count)
		})
	}

}

func (s *ComplianceScanConfigServiceTestSuite) TestGetComplianceScanConfiguration() {
	allAccessContext := sac.WithAllAccess(context.Background())
	createdTime := timestamp.Now().GogoProtobuf()
	lastUpdatedTime := timestamp.Now().GogoProtobuf()

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
						CreatedTime:     createdTime,
						LastUpdatedTime: lastUpdatedTime,
						ModifiedBy:      storageRequester,
						Description:     "test-description",
					}, true, nil).Times(1)

				s.scanConfigDatastore.EXPECT().GetScanConfigClusterStatus(allAccessContext, uuid.NewDummy().String()).Return([]*storage.ComplianceOperatorClusterScanConfigStatus{
					{
						ClusterId:    fixtureconsts.Cluster1,
						ClusterName:  mockClusterName,
						ScanConfigId: uuid.NewDummy().String(),
						Errors:       []string{"Error 1", "Error 2", "Error 3"},
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
			s.Require().Equal(tc.expectedResp, config)
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

func getTestAPIStatusRec(createdTime, lastUpdatedTime *types.Timestamp) *apiV2.ComplianceScanConfigurationStatus {
	return &apiV2.ComplianceScanConfigurationStatus{
		Id:       uuid.NewDummy().String(),
		ScanName: "test-scan",
		ScanConfig: &apiV2.BaseComplianceScanConfigurationSettings{
			OneTimeScan:  false,
			Profiles:     []string{"ocp4-cis"},
			ScanSchedule: defaultAPISchedule,
			Description:  "test-description",
		},
		ClusterStatus: []*apiV2.ClusterScanStatus{
			{
				ClusterId:   fixtureconsts.Cluster1,
				ClusterName: mockClusterName,
				Errors:      []string{"Error 1", "Error 2", "Error 3"},
			},
		},
		CreatedTime:     createdTime,
		LastUpdatedTime: lastUpdatedTime,
		ModifiedBy:      apiRequester,
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
