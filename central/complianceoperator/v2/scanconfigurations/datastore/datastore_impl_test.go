//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	scanStatusStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/scanconfigstatus/store/postgres"
	configStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
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
	cluster3ReadWriteCtx        = "cluster3ReadWriteCtx"
	complianceWriteNoClusterCtx = "complianceWriteNoClusterCtx"
	clusterWriteNoComplianceCtx = "clusterWriteNoComplianceCtx"
	noAccessCtx                 = "noAccessCtx"
)

var (
	log = logging.LoggerForModule()
)

func TestComplianceScanConfigDataStore(t *testing.T) {
	suite.Run(t, new(complianceScanConfigDataStoreTestSuite))
}

type complianceScanConfigDataStoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	testContexts map[string]context.Context

	dataStore     DataStore
	db            *pgtest.TestPostgres
	storage       configStore.Store
	statusStorage scanStatusStore.Store

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

	clusterDatastore, err := clusterDS.GetTestPostgresDataStore(s.T(), s.db)
	s.Require().NoError(err)

	clusterID1, cluster1AddErr := clusterDatastore.AddCluster(sac.WithAllAccess(context.Background()), &storage.Cluster{
		Name:      mockClusterName,
		MainImage: "4.3.0",
	})
	s.clusterID1 = clusterID1
	s.Require().NoError(cluster1AddErr)
	clusterID2, cluster2AddErr := clusterDatastore.AddCluster(sac.WithAllAccess(context.Background()), &storage.Cluster{
		Name:      "mock-cluster-2",
		MainImage: "4.3.0",
	})
	s.clusterID2 = clusterID2
	s.Require().NoError(cluster2AddErr)

	s.storage = configStore.New(s.db)
	s.statusStorage = scanStatusStore.New(s.db)

	s.dataStore = New(s.storage, s.statusStorage, clusterDatastore)

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

func (s *complianceScanConfigDataStoreTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *complianceScanConfigDataStoreTestSuite) TestGetScanConfiguration() {
	configID := uuid.NewV4().String()

	scanConfig := s.getTestRec(mockScanName)
	scanConfig.Id = configID

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig))

	for _, cluster := range scanConfig.Clusters {
		s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID, cluster.ClusterId, "testing status"))
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
		s.Require().Equal(tc.expectedResponse, foundConfig)
	}
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
		s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID1, cluster.ClusterId, "testing status"))
	}
	for _, cluster := range scanConfig2.Clusters {
		s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID2, cluster.ClusterId, "testing status"))
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
		s.Require().Equal(tc.expectedResponses, scanConfigs)
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

func (s *complianceScanConfigDataStoreTestSuite) TestScanConfigurationExists() {
	configID := uuid.NewV4().String()

	scanConfig := s.getTestRec(mockScanName)
	scanConfig.Id = configID

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.testContexts[unrestrictedReadWriteCtx], scanConfig))

	found, err := s.dataStore.ScanConfigurationExists(s.testContexts[unrestrictedReadCtx], mockScanName)
	s.Require().NoError(err)
	s.Require().True(found)

	// Retrieve a record that does not exist
	found, err = s.dataStore.ScanConfigurationExists(s.testContexts[unrestrictedReadCtx], "DOES NOT EXIST")
	s.Require().NoError(err)
	s.Require().False(found)

	// Try to retrieve a record with no access
	found, err = s.dataStore.ScanConfigurationExists(s.testContexts[noAccessCtx], mockScanName)
	s.Require().NoError(err)
	s.Require().False(found)
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
			s.Require().Equal(scanConfig, foundConfig)

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
		s.Require().Equal(scanConfig, foundConfig)

		// Add Scan config status
		for _, cluster := range tc.clusters {
			s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID, cluster, "testing status"))
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
	s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID1, s.clusterID1, "testing status"))
	s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID1, s.clusterID2, "testing status"))
	s.Require().NoError(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], configID2, s.clusterID1, "testing status"))

	clusterStatuses, err := s.dataStore.GetScanConfigClusterStatus(s.testContexts[unrestrictedReadCtx], configID1)
	s.Require().NoError(err)
	s.Require().Equal(2, len(clusterStatuses))

	// Try to add one with no existing scan config
	s.Require().NotNil(s.dataStore.UpdateClusterStatus(s.testContexts[unrestrictedReadWriteCtx], uuid.NewDummy().String(), fixtureconsts.Cluster1, "testing status"))
	clusterStatuses, err = s.dataStore.GetScanConfigClusterStatus(s.testContexts[unrestrictedReadCtx], uuid.NewDummy().String())
	s.Require().NoError(err)
	s.Require().Equal(0, len(clusterStatuses))

	// No access to read clusters so should return an error
	s.Require().Error(s.dataStore.UpdateClusterStatus(s.testContexts[complianceWriteNoClusterCtx], configID1, fixtureconsts.Cluster1, "testing status"))
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
