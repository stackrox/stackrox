//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	scanStatusStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/scanconfigstatus/store/postgres"
	configStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	mockClusterName = "mock-cluster"
	mockScanName    = "mock-scan"

	unrestrictedReadCtx         = "unrestrictedReadCtx"
	unrestrictedReadWriteCtx    = "unrestrictedReadWriteCtx"
	cluster1ReadCtx             = "cluster1ReadCtx"
	cluster1ReadWriteCtx        = "cluster1ReadWriteCtx"
	cluster2ReadWriteCtx        = "cluster2ReadWriteCtx"
	cluster3ReadWriteCtx        = "cluster3ReadWriteCtx"
	complianceWriteNoClusterCtx = "complianceWriteNoClusterCtx"
	clusterWriteNoComplianceCtx = "clusterWriteNoComplianceCtx"
	noAccessCtx                 = "noAccessCtx"

	maxPaginationLimit = 1000
)

func TestComplianceScanConfigDataStore(t *testing.T) {
	suite.Run(t, new(complianceScanConfigDataStoreTestSuite))
}

type complianceScanConfigDataStoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	testContexts map[string]context.Context

	dataStore        DataStore
	db               *pgtest.TestPostgres
	storage          configStore.Store
	statusStorage    scanStatusStore.Store
	scanConfigDSMock *scanConfigMocks.MockDataStore

	clusterID1 string
	clusterID2 string
}

func (s *complianceScanConfigDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *complianceScanConfigDataStoreTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.db = pgtest.ForT(s.T())

	s.storage = configStore.New(s.db)
	s.statusStorage = scanStatusStore.New(s.db)
	s.scanConfigDSMock = scanConfigMocks.NewMockDataStore(s.mockCtrl)

	s.dataStore = New(s.storage, s.statusStorage, s.db.DB)
	s.clusterID1 = testconsts.Cluster1
	s.clusterID2 = testconsts.Cluster2

	// Setup SAC contexts
	s.testContexts = make(map[string]context.Context, 0)
	resourceHandles := []permissions.ResourceHandle{resources.Compliance, resources.Cluster}
	s.testContexts[unrestrictedReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...)))

	s.testContexts[cluster1ReadCtx] =
		sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(s.clusterID1)))

	s.testContexts[cluster1ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(s.clusterID1)))

	s.testContexts[cluster2ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(s.clusterID2)))

	s.testContexts[cluster3ReadWriteCtx] =
		sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resourceHandles...),
				sac.ClusterScopeKeys(testconsts.Cluster3)))

	s.testContexts[unrestrictedReadCtx] = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resourceHandles...)))

	s.testContexts[complianceWriteNoClusterCtx] = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))

	s.testContexts[clusterWriteNoComplianceCtx] = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster)))

	s.testContexts[noAccessCtx] = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
}

func (s *complianceScanConfigDataStoreTestSuite) TestGetScanConfiguration() {
	configID := uuid.NewV4().String()

	scanConfig := s.getTestRec(mockScanName)
	scanConfig.Id = configID

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig))

	for _, cluster := range scanConfig.Clusters {
		s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID, cluster.ClusterId, "testing status", ""))
	}

	testCases := []struct {
		desc             string
		scanConfigID     string
		testContext      context.Context
		expectedResponse *storage.ComplianceOperatorScanConfigurationV2
		isErrorTest      bool
	}{
		{
			desc:             "Successful get - Full access",
			scanConfigID:     configID,
			testContext:      s.testContexts[unrestrictedReadWriteCtx],
			expectedResponse: scanConfig,
			isErrorTest:      false,
		},
		{
			desc:             "Successful get multiple clusters - Full access",
			scanConfigID:     configID,
			testContext:      s.testContexts[unrestrictedReadWriteCtx],
			expectedResponse: scanConfig,
			isErrorTest:      false,
		},
		{
			desc:             "No cluster access",
			scanConfigID:     configID,
			testContext:      s.testContexts[noAccessCtx],
			expectedResponse: nil,
			isErrorTest:      false,
		},
		{
			desc:             "Access to cluster 1 but config is for cluster 1 and cluster 2",
			scanConfigID:     configID,
			testContext:      s.testContexts[cluster1ReadWriteCtx],
			expectedResponse: nil,
			isErrorTest:      true,
		},
		{
			desc:             "Access to cluster not related to the scan config",
			scanConfigID:     configID,
			testContext:      s.testContexts[cluster3ReadWriteCtx],
			expectedResponse: nil,
			isErrorTest:      true,
		},
	}

	for _, tc := range testCases {
		log.Info(tc.desc)
		foundConfig, found, err := s.dataStore.GetScanConfiguration(tc.testContext, tc.scanConfigID)
		s.Require().NoError(err)
		if tc.expectedResponse != nil {
			s.Require().True(found)
		} else {
			s.Require().False(found)
		}
		protoassert.Equal(s.T(), tc.expectedResponse, foundConfig)
	}
}

func (s *complianceScanConfigDataStoreTestSuite) TestRemoveClusterFromScanConfig() {
	configID1 := uuid.NewV4().String()
	scanConfig1 := s.getTestRec(mockScanName)
	scanConfig1.Id = configID1
	configID2 := uuid.NewV4().String()
	scanConfig2 := s.getTestRec("mock-scan-config-2")
	scanConfig2.Id = configID2
	scanConfig2.Clusters = []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
		{
			ClusterId: s.clusterID2,
		},
	}
	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig1))
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig2))

	for _, cluster := range scanConfig1.Clusters {
		s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID1, cluster.ClusterId, "testing status", ""))
	}

	for _, cluster := range scanConfig2.Clusters {
		s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID2, cluster.ClusterId, "testing status", ""))
	}

	err := s.dataStore.RemoveClusterFromScanConfig(s.testContexts[unrestrictedReadWriteCtx], scanConfig1.Clusters[0].GetClusterId())
	s.Require().NoError(err)

	newscanConfig, exists, err := s.dataStore.GetScanConfiguration(s.testContexts[unrestrictedReadWriteCtx], scanConfig1.GetId())
	s.Require().NoError(err)
	s.Require().True(exists, "scan config not found")
	s.Require().Less(len(newscanConfig.GetClusters()), len(scanConfig1.GetClusters()))

	scanConfigStatus, err := s.dataStore.GetScanConfigClusterStatus(s.testContexts[unrestrictedReadWriteCtx], scanConfig1.GetId())
	s.Require().NoError(err)
	s.Require().Equal(len(newscanConfig.GetClusters()), len(scanConfigStatus))

	err = s.dataStore.RemoveClusterFromScanConfig(s.testContexts[unrestrictedReadWriteCtx], scanConfig2.Clusters[0].GetClusterId())
	s.Require().NoError(err)
	newscanConfig, exists, err = s.dataStore.GetScanConfiguration(s.testContexts[unrestrictedReadWriteCtx], scanConfig2.GetId())
	s.Require().NoError(err)
	s.Require().True(exists, "scan config not found")
	s.Require().Less(len(newscanConfig.GetClusters()), len(scanConfig2.GetClusters()))
}

func (s *complianceScanConfigDataStoreTestSuite) TestGetScanConfigurations() {
	configID1 := uuid.NewV4().String()
	configID2 := uuid.NewV4().String()

	scanConfig1 := s.getTestRec(mockScanName)
	scanConfig1.Id = configID1
	scanConfig2 := s.getTestRec("mock-scan-config-2")
	scanConfig2.Id = configID2

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig1))
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig2))

	for _, cluster := range scanConfig1.Clusters {
		s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID1, cluster.ClusterId, "testing status", ""))
	}
	for _, cluster := range scanConfig2.Clusters {
		s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID2, cluster.ClusterId, "testing status", ""))
	}

	testCases := []struct {
		desc              string
		query             *v1.Query
		testContext       context.Context
		expectedResponses []*storage.ComplianceOperatorScanConfigurationV2
	}{
		{
			desc: "Successful get - Full access",
			query: search.NewQueryBuilder().
				AddExactMatches(search.ComplianceOperatorScanConfigName, mockScanName).ProtoQuery(),
			testContext:       s.testContexts[unrestrictedReadWriteCtx],
			expectedResponses: []*storage.ComplianceOperatorScanConfigurationV2{scanConfig1},
		},
		{
			desc: "Successful get multiple configs - Full access",
			query: search.NewQueryBuilder().
				AddExactMatches(search.ComplianceOperatorScanConfigName, mockScanName).
				AddExactMatches(search.ComplianceOperatorScanConfigName, "mock-scan-config-2").ProtoQuery(),
			testContext:       s.testContexts[unrestrictedReadWriteCtx],
			expectedResponses: []*storage.ComplianceOperatorScanConfigurationV2{scanConfig1, scanConfig2},
		},
		{
			desc: "No cluster access",
			query: search.NewQueryBuilder().
				AddExactMatches(search.ComplianceOperatorScanConfigName, mockScanName).ProtoQuery(),
			testContext:       s.testContexts[noAccessCtx],
			expectedResponses: nil,
		},
		{
			desc: "Access to cluster 1 but config is for cluster 1 and cluster 2",
			query: search.NewQueryBuilder().
				AddExactMatches(search.ComplianceOperatorScanConfigName, mockScanName).ProtoQuery(),
			testContext:       s.testContexts[cluster1ReadWriteCtx],
			expectedResponses: nil,
		},
		{
			desc: "Access to cluster not related to the scan config",
			query: search.NewQueryBuilder().
				AddExactMatches(search.ComplianceOperatorScanConfigName, mockScanName).ProtoQuery(),
			testContext:       s.testContexts[cluster3ReadWriteCtx],
			expectedResponses: nil,
		},
		{
			desc: "Full access, config does not exist",
			query: search.NewQueryBuilder().
				AddExactMatches(search.ComplianceOperatorScanConfigName, "DOESNOTEXIST").ProtoQuery(),
			testContext:       s.testContexts[unrestrictedReadWriteCtx],
			expectedResponses: nil,
		},
	}

	for _, tc := range testCases {
		log.Info(tc.desc)
		scanConfigs, err := s.dataStore.GetScanConfigurations(tc.testContext, tc.query)
		s.Require().NoError(err)
		s.Require().Equal(len(tc.expectedResponses), len(scanConfigs))
		protoassert.SlicesEqual(s.T(), tc.expectedResponses, scanConfigs)
	}
}

func (s *complianceScanConfigDataStoreTestSuite) TestGetScanConfigurationsCount() {
	configID := uuid.NewV4().String()

	scanConfig := s.getTestRec(mockScanName)
	scanConfig.Id = configID

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig))

	q := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanConfigName, mockScanName).ProtoQuery()
	count, err := s.dataStore.CountScanConfigurations(s.testContexts[unrestrictedReadCtx], q)
	s.Require().NoError(err)
	s.Require().Equal(1, count)

	// Get count without access
	count, err = s.dataStore.CountScanConfigurations(s.testContexts[noAccessCtx], q)
	s.Require().NoError(err)
	s.Require().Equal(0, count)
}

func (s *complianceScanConfigDataStoreTestSuite) TestGetScanConfigurationByName() {
	configID := uuid.NewV4().String()

	scanConfig := s.getTestRec(mockScanName)
	scanConfig.Id = configID

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig))

	foundConfig, err := s.dataStore.GetScanConfigurationByName(s.testContexts[unrestrictedReadCtx], mockScanName)
	s.Require().NoError(err)
	protoassert.Equal(s.T(), scanConfig, foundConfig)

	// Retrieve a record that does not exist
	foundConfig, err = s.dataStore.GetScanConfigurationByName(s.testContexts[unrestrictedReadCtx], "DOES NOT EXIST")
	s.Require().NoError(err)
	s.Require().Nil(foundConfig)

	// Try to retrieve a record with no access
	foundConfig, err = s.dataStore.GetScanConfigurationByName(s.testContexts[noAccessCtx], mockScanName)
	s.Require().NoError(err)
	s.Require().Nil(foundConfig)
}

func (s *complianceScanConfigDataStoreTestSuite) TestScanConfigurationProfileExists() {
	configID := uuid.NewV4().String()

	scanConfig := s.getTestRec(mockScanName)
	scanConfig.Id = configID

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig))

	// Add testCases for different combinations of profiles and clusters
	testCases := []struct {
		desc        string
		configID    string
		profiles    []string
		clusters    []string
		isErrorTest bool
	}{
		{
			desc:        "Successful get - no deplicate profiles",
			configID:    uuid.NewV4().String(),
			profiles:    []string{"no-match-profile", "ocp4-cis-node"},
			clusters:    []string{s.clusterID1},
			isErrorTest: false,
		},
		{
			desc:        "Successful get - duplicate profiles",
			configID:    uuid.NewV4().String(),
			profiles:    []string{"no-match-profile", "ocp4-cis"},
			clusters:    []string{s.clusterID1},
			isErrorTest: true,
		},
		{
			desc:        "Successful get - duplicate profiles, multiple clusters",
			configID:    uuid.NewV4().String(),
			profiles:    []string{"no-match-profile", "ocp4-cis"},
			clusters:    []string{s.clusterID1, s.clusterID2},
			isErrorTest: true,
		},
		{
			desc:        "Successful get - duplicate profiles, multiple clusters, multiple profiles",
			configID:    uuid.NewV4().String(),
			profiles:    []string{"no-match-profile", "ocp4-cis", "ocp4-cis-node"},
			clusters:    []string{s.clusterID1, s.clusterID2},
			isErrorTest: true,
		},
		{
			desc:        "Successful get - one duplicate profile with version string",
			configID:    uuid.NewV4().String(),
			profiles:    []string{"ocp4-cis-1-4-0"},
			clusters:    []string{s.clusterID1},
			isErrorTest: false,
		},
		{
			desc:        "Successful get - updating existing config",
			configID:    configID,
			profiles:    []string{"no-match-profile", "ocp4-cis"},
			clusters:    []string{s.clusterID1},
			isErrorTest: false,
		},
		{
			desc:        "Successful get - updating existing config",
			configID:    configID,
			profiles:    []string{"no-match-profile", "ocp4-cis", "ocp4-cis-1-4-0"},
			clusters:    []string{s.clusterID1},
			isErrorTest: false,
		},
	}

	for _, tc := range testCases {
		log.Info(tc.desc)
		err := s.dataStore.ScanConfigurationProfileExists(s.testContexts[unrestrictedReadCtx], tc.configID, tc.profiles, tc.clusters)
		if tc.isErrorTest {
			s.Require().Error(err)
		} else {
			s.Require().NoError(err)
		}
	}

}

func (s *complianceScanConfigDataStoreTestSuite) TestUpsertScanConfiguration() {
	configID := uuid.NewV4().String()

	// This config has cluster 1 and cluster 2
	scanConfig := s.getTestRec(mockScanName)
	scanConfig.Id = configID

	testCases := []struct {
		desc        string
		scanName    string
		testContext context.Context
		isErrorTest bool
	}{
		{
			desc:        "Successful update - Full access",
			scanName:    "case-1",
			testContext: s.testContexts[unrestrictedReadWriteCtx],
			isErrorTest: false,
		},
		{
			desc:        "Successful update multiple clusters - Full access",
			scanName:    "case-2",
			testContext: s.testContexts[unrestrictedReadWriteCtx],
			isErrorTest: false,
		},
		{
			desc:        "No cluster access",
			scanName:    "case-3",
			testContext: s.testContexts[noAccessCtx],
			isErrorTest: true,
		},
		{
			desc:        "Multiple clusters config -- only access to one",
			scanName:    "case-4",
			testContext: s.testContexts[cluster1ReadWriteCtx],
			isErrorTest: true,
		},
	}

	for _, tc := range testCases {
		log.Info(tc.desc)

		err := s.dataStore.UpsertScanConfiguration(tc.testContext, scanConfig)
		if tc.isErrorTest {
			s.Require().Error(err)
		} else {
			s.Require().NoError(err)

			// Verify we can get what we just added
			foundConfig, found, err := s.dataStore.GetScanConfiguration(s.testContexts[unrestrictedReadCtx], configID)
			s.Require().NoError(err)
			s.Require().True(found)
			protoassert.Equal(s.T(), scanConfig, foundConfig)

			// Clean up for the next run
			_, err = s.dataStore.DeleteScanConfiguration(s.testContexts[unrestrictedReadWriteCtx], configID)
			s.Require().NoError(err)
		}
	}
}

func (s *complianceScanConfigDataStoreTestSuite) TestDeleteScanConfiguration() {
	testCases := []struct {
		desc        string
		scanName    string
		testContext context.Context
		clusters    []string
		isErrorTest bool
	}{
		{
			desc:        "Successful delete - Full access",
			scanName:    "case-1",
			testContext: s.testContexts[unrestrictedReadWriteCtx],
			clusters:    []string{s.clusterID1},
			isErrorTest: false,
		},
		{
			desc:        "Successful delete multiple clusters - Full access",
			scanName:    "case-2",
			testContext: s.testContexts[unrestrictedReadWriteCtx],
			clusters:    []string{s.clusterID1, s.clusterID2},
			isErrorTest: false,
		},
		{
			desc:        "No cluster access",
			scanName:    "case-3",
			testContext: s.testContexts[cluster1ReadWriteCtx],
			clusters:    []string{s.clusterID2},
			isErrorTest: true,
		},
		{
			desc:        "Multiple clusters only access to one",
			scanName:    "case-4",
			testContext: s.testContexts[cluster1ReadWriteCtx],
			clusters:    []string{s.clusterID1, s.clusterID2},
			isErrorTest: true,
		},
	}

	for _, tc := range testCases {
		log.Info(tc.desc)
		configID := uuid.NewV4().String()

		scanConfig := s.getTestRec(tc.scanName)
		scanConfig.Id = configID

		err := s.dataStore.UpsertScanConfiguration(s.testContexts[unrestrictedReadWriteCtx], scanConfig)
		s.Require().NoError(err)

		// Verify we can get what we just added
		foundConfig, found, err := s.dataStore.GetScanConfiguration(s.testContexts[unrestrictedReadCtx], configID)
		s.Require().NoError(err)
		s.Require().True(found)
		protoassert.Equal(s.T(), scanConfig, foundConfig)

		// Add Scan config status
		for _, cluster := range tc.clusters {
			s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID, cluster, "testing status", ""))
		}

		// Ensure we find status
		clusterStatuses, err := s.dataStore.GetScanConfigClusterStatus(s.testContexts[unrestrictedReadCtx], configID)
		s.Require().NoError(err)
		s.Require().Equal(len(tc.clusters), len(clusterStatuses))

		// Now delete it
		scanConfigName, err := s.dataStore.DeleteScanConfiguration(tc.testContext, configID)
		// If it is not an error case, make sure the record was deleted.
		if !tc.isErrorTest {
			s.Require().NoError(err)
			s.Require().Equal(tc.scanName, scanConfigName)

			// Verify it no longer exists
			foundConfig, found, err = s.dataStore.GetScanConfiguration(s.testContexts[unrestrictedReadCtx], configID)
			s.Require().NoError(err)
			s.Require().False(found)
			s.Require().Nil(foundConfig)

			// cluster status should also be deleted
			clusterStatuses, err = s.dataStore.GetScanConfigClusterStatus(s.testContexts[unrestrictedReadCtx], configID)
			s.Require().NoError(err)
			s.Require().Equal(0, len(clusterStatuses))
		} else {
			s.Require().Error(err, "access to resource denied")
		}
	}
}

func (s *complianceScanConfigDataStoreTestSuite) TestClusterStatus() {
	configID1 := uuid.NewV4().String()
	scanConfig1 := s.getTestRec(mockScanName)
	scanConfig1.Id = configID1

	configID2 := uuid.NewV4().String()
	scanConfig2 := s.getTestRec("mockScan2")
	scanConfig2.Id = configID2

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig1))
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig2))

	// Add Scan config status
	s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID1, s.clusterID1, "testing status", ""))
	s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID1, s.clusterID2, "testing status", ""))
	s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID2, s.clusterID1, "testing status", ""))

	clusterStatuses, err := s.dataStore.GetScanConfigClusterStatus(s.testContexts[unrestrictedReadCtx], configID1)
	s.Require().NoError(err)
	s.Require().Equal(2, len(clusterStatuses))

	// Try to add one with no existing scan config
	s.Require().NotNil(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], uuid.NewDummy().String(), fixtureconsts.Cluster1, "testing status", ""))
	clusterStatuses, err = s.dataStore.GetScanConfigClusterStatus(s.testContexts[unrestrictedReadCtx], uuid.NewDummy().String())
	s.Require().NoError(err)
	s.Require().Equal(0, len(clusterStatuses))

	// No access to read clusters so should return an error
	s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[complianceWriteNoClusterCtx], configID1, fixtureconsts.Cluster1, "testing status", ""))
}

func (s *complianceScanConfigDataStoreTestSuite) TestGetProfilesNames() {
	configID1 := uuid.NewV4().String()
	scanConfig1 := s.getTestRec(mockScanName)
	scanConfig1.Id = configID1
	scanConfig1.Profiles = []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
		{
			ProfileName: "ocp4-cis",
		},
		{
			ProfileName: "rhcos-moderate",
		},
		{
			ProfileName: "a-rhcos-moderate",
		},
	}

	configID2 := uuid.NewV4().String()
	scanConfig2 := s.getTestRec("mockScan2")
	scanConfig2.Id = configID2
	scanConfig2.Profiles = []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
		{
			ProfileName: "ocp4-cis",
		},
		{
			ProfileName: "rhcos-moderate",
		},
	}

	configID3 := uuid.NewV4().String()
	scanConfig3 := s.getTestRec("mockScan3")
	scanConfig3.Id = configID3
	scanConfig3.Profiles = []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
		{
			ProfileName: "yet-another-profile",
		},
	}
	scanConfig3.Clusters = []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
		{
			ClusterId: s.clusterID1,
		},
	}

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig1))
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig2))
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig3))

	testCases := []struct {
		desc           string
		query          *v1.Query
		countQuery     *v1.Query
		testContext    context.Context
		expectedRecord []string
		expectedCount  int
	}{
		{
			desc:           "Full access",
			query:          nil,
			testContext:    s.testContexts[unrestrictedReadCtx],
			expectedRecord: []string{"a-rhcos-moderate", "ocp4-cis", "rhcos-moderate", "yet-another-profile"},
			expectedCount:  4,
		},
		{
			desc:           "Full access paging for record 2",
			query:          search.NewQueryBuilder().WithPagination(search.NewPagination().Limit(1).Offset(2)).ProtoQuery(),
			countQuery:     search.NewQueryBuilder().ProtoQuery(),
			testContext:    s.testContexts[unrestrictedReadCtx],
			expectedRecord: []string{"ocp4-cis"},
			expectedCount:  4, // because of paging, total count will be 4
		},
		{
			desc:           "Cluster 1 - Only cluster 3 access",
			query:          search.NewQueryBuilder().AddExactMatches(search.ClusterID, testconsts.Cluster1).WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			countQuery:     search.NewQueryBuilder().AddExactMatches(search.ClusterID, testconsts.Cluster1).ProtoQuery(),
			testContext:    s.testContexts[cluster3ReadWriteCtx],
			expectedRecord: nil,
			expectedCount:  0,
		},
		{
			desc:           "Cluster 2 query - Only cluster 2 access",
			query:          search.NewQueryBuilder().AddExactMatches(search.ClusterID, testconsts.Cluster2).WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			countQuery:     search.NewQueryBuilder().AddExactMatches(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			testContext:    s.testContexts[cluster2ReadWriteCtx],
			expectedRecord: []string{"a-rhcos-moderate", "ocp4-cis", "rhcos-moderate"},
			expectedCount:  3,
		},
		{
			desc:           "Cluster 1 and 2 query - Only cluster 2 access",
			query:          search.NewQueryBuilder().AddExactMatches(search.ClusterID, testconsts.Cluster2).WithPagination(search.NewPagination().Limit(maxPaginationLimit)).ProtoQuery(),
			countQuery:     search.NewQueryBuilder().AddExactMatches(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			testContext:    s.testContexts[cluster2ReadWriteCtx],
			expectedRecord: []string{"a-rhcos-moderate", "ocp4-cis", "rhcos-moderate"},
			expectedCount:  3,
		},
	}

	for _, tc := range testCases {
		log.Info(tc.desc)
		profiles, err := s.dataStore.GetProfilesNames(tc.testContext, tc.query)
		s.Require().NoError(err)
		if tc.expectedRecord == nil {
			s.Require().Equal(0, len(profiles))
		} else {
			s.Require().ElementsMatch(tc.expectedRecord, profiles)
		}
		count, err := s.dataStore.CountDistinctProfiles(tc.testContext, tc.countQuery)
		s.Require().NoError(err)
		s.Require().Equal(tc.expectedCount, count)
	}
}

func (s *complianceScanConfigDataStoreTestSuite) getTestRec(scanName string) *storage.ComplianceOperatorScanConfigurationV2 {
	return &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName:         scanName,
		AutoApplyRemediations:  false,
		AutoUpdateRemediations: false,
		OneTimeScan:            false,
		Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
			{
				ProfileName: "ocp4-cis",
			},
		},
		StrictNodeScan: false,
		Description:    "test-description",
		Clusters: []*storage.ComplianceOperatorScanConfigurationV2_Cluster{
			{
				ClusterId: s.clusterID1,
			},
			{
				ClusterId: s.clusterID2,
			},
		},
	}
}
