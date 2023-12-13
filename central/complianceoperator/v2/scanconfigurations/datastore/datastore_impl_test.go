//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	clusterPG "github.com/stackrox/rox/central/cluster/store/cluster/postgres"
	configSearch "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/search"
	scanStatusStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/scanconfigstatus/store/postgres"
	configStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	mockClusterName = "mock-cluster"
	mockScanName    = "mock-scan"
)

func TestComplianceScanConfigDataStore(t *testing.T) {
	suite.Run(t, new(complianceScanConfigDataStoreTestSuite))
}

type complianceScanConfigDataStoreTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	hasReadCtx           context.Context
	hasWriteCtx          context.Context
	noAccessCtx          context.Context
	hasWriteNoClusterCtx context.Context

	dataStore     DataStore
	db            *pgtest.TestPostgres
	storage       configStore.Store
	statusStorage scanStatusStore.Store
	search        configSearch.Searcher
}

func (s *complianceScanConfigDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *complianceScanConfigDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.ComplianceOperator, resources.Cluster)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ComplianceOperator, resources.Cluster)))
	s.hasWriteNoClusterCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ComplianceOperator)))
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	s.mockCtrl = gomock.NewController(s.T())
	s.db = pgtest.ForT(s.T())

	clusterDatastore, err := clusterDS.GetTestPostgresDataStore(s.T(), s.db)
	s.Require().NoError(err)

	clusterStore := clusterPG.New(s.db)
	s.Require().NoError(clusterStore.Upsert(sac.WithAllAccess(context.Background()), &storage.Cluster{
		Id:   fixtureconsts.Cluster1,
		Name: mockClusterName,
	}))
	s.Require().NoError(clusterStore.Upsert(sac.WithAllAccess(context.Background()), &storage.Cluster{
		Id:   fixtureconsts.Cluster2,
		Name: "mock-cluster-2",
	}))

	s.storage = configStore.New(s.db)
	s.statusStorage = scanStatusStore.New(s.db)
	indexer := configStore.NewIndexer(s.db)
	configStorage := configStore.New(s.db)
	s.search = configSearch.New(configStorage, indexer)

	s.dataStore = New(s.storage, s.statusStorage, clusterDatastore, s.search)
}

func (s *complianceScanConfigDataStoreTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *complianceScanConfigDataStoreTestSuite) TestGetScanConfiguration() {
	configID := uuid.NewV4().String()

	scanConfig := getTestRec(mockScanName)
	scanConfig.Id = configID

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, scanConfig))

	foundConfig, found, err := s.dataStore.GetScanConfiguration(s.hasReadCtx, configID)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(scanConfig, foundConfig)

	// Retrieve a record that does not exist
	foundConfig, found, err = s.dataStore.GetScanConfiguration(s.hasReadCtx, uuid.NewV4().String())
	s.Require().NoError(err)
	s.Require().False(found)
	s.Require().Nil(foundConfig)

	// Try to retrieve with improper permissions
	foundConfig, found, err = s.dataStore.GetScanConfiguration(s.noAccessCtx, configID)
	s.Require().Nil(err)
	s.Require().False(found)
	s.Require().Nil(foundConfig)
}

func (s *complianceScanConfigDataStoreTestSuite) TestGetScanConfigurations() {
	configID := uuid.NewV4().String()

	scanConfig := getTestRec(mockScanName)
	scanConfig.Id = configID

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, scanConfig))

	scanConfigs, err := s.dataStore.GetScanConfigurations(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanName, mockScanName).ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(1, len(scanConfigs))
	s.Require().Equal(scanConfig, scanConfigs[0])

	scanConfigs, err = s.dataStore.GetScanConfigurations(s.hasReadCtx, search.NewQueryBuilder().
		AddExactMatches(search.ComplianceOperatorScanName, "DOESNOTEXIST").ProtoQuery())
	s.Require().NoError(err)
	s.Require().Equal(0, len(scanConfigs))
}

func (s *complianceScanConfigDataStoreTestSuite) TestGetScanConfigurationsCount() {
	configID := uuid.NewV4().String()

	scanConfig := getTestRec(mockScanName)
	scanConfig.Id = configID

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, scanConfig))

	q := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorScanName, mockScanName).ProtoQuery()
	count, err := s.dataStore.CountScanConfigurations(s.hasReadCtx, q)
	s.Require().NoError(err)
	s.Require().Equal(1, count)
}

func (s *complianceScanConfigDataStoreTestSuite) TestScanConfigurationExists() {
	configID := uuid.NewV4().String()

	scanConfig := getTestRec(mockScanName)
	scanConfig.Id = configID

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, scanConfig))

	found, err := s.dataStore.ScanConfigurationExists(s.hasReadCtx, mockScanName)
	s.Require().NoError(err)
	s.Require().True(found)

	// Retrieve a record that does not exist
	found, err = s.dataStore.ScanConfigurationExists(s.hasReadCtx, "DOES NOT EXIST")
	s.Require().NoError(err)
	s.Require().False(found)
}

func (s *complianceScanConfigDataStoreTestSuite) TestUpsertScanConfiguration() {
	configID := uuid.NewV4().String()

	scanConfig := getTestRec(mockScanName)
	scanConfig.Id = configID

	err := s.dataStore.UpsertScanConfiguration(s.hasWriteCtx, scanConfig)
	s.Require().NoError(err)

	// Verify we can get what we just added
	foundConfig, found, err := s.dataStore.GetScanConfiguration(s.hasReadCtx, configID)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(scanConfig, foundConfig)
}

func (s *complianceScanConfigDataStoreTestSuite) TestDeleteScanConfiguration() {
	configID := uuid.NewV4().String()

	scanConfig := getTestRec(mockScanName)
	scanConfig.Id = configID

	err := s.dataStore.UpsertScanConfiguration(s.hasWriteCtx, scanConfig)
	s.Require().NoError(err)

	// Verify we can get what we just added
	foundConfig, found, err := s.dataStore.GetScanConfiguration(s.hasReadCtx, configID)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Require().Equal(scanConfig, foundConfig)

	// Now delete it
	s.Require().NoError(s.dataStore.DeleteScanConfiguration(s.hasWriteCtx, configID))

	// Verify it no longer exists
	foundConfig, found, err = s.dataStore.GetScanConfiguration(s.hasReadCtx, configID)
	s.Require().NoError(err)
	s.Require().False(found)
	s.Require().Nil(foundConfig)

	// Delete a non-existing one
	err = s.dataStore.DeleteScanConfiguration(s.hasWriteCtx, uuid.NewV4().String())
	s.Require().NoError(err)
}

func (s *complianceScanConfigDataStoreTestSuite) TestClusterStatus() {
	configID1 := uuid.NewV4().String()
	scanConfig1 := getTestRec(mockScanName)
	scanConfig1.Id = configID1

	configID2 := uuid.NewV4().String()
	scanConfig2 := getTestRec("mockScan2")
	scanConfig2.Id = configID2

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, scanConfig1))
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, scanConfig2))

	// Add Scan config status
	s.Require().NoError(s.dataStore.UpdateClusterStatus(s.hasWriteCtx, configID1, fixtureconsts.Cluster1, "testing status"))
	s.Require().NoError(s.dataStore.UpdateClusterStatus(s.hasWriteCtx, configID1, fixtureconsts.Cluster2, "testing status"))
	s.Require().NoError(s.dataStore.UpdateClusterStatus(s.hasWriteCtx, configID2, fixtureconsts.Cluster1, "testing status"))

	clusterStatuses, err := s.dataStore.GetScanConfigClusterStatus(s.hasReadCtx, configID1)
	s.Require().NoError(err)
	s.Require().Equal(2, len(clusterStatuses))

	// Try to add one with no existing scan config
	s.Require().NotNil(s.dataStore.UpdateClusterStatus(s.hasWriteCtx, uuid.NewDummy().String(), fixtureconsts.Cluster1, "testing status"))
	clusterStatuses, err = s.dataStore.GetScanConfigClusterStatus(s.hasReadCtx, uuid.NewDummy().String())
	s.Require().NoError(err)
	s.Require().Equal(0, len(clusterStatuses))

	// No access to read clusters so should return an error
	s.Require().Error(s.dataStore.UpdateClusterStatus(s.hasWriteNoClusterCtx, configID1, fixtureconsts.Cluster1, "testing status"))
}

func getTestRec(scanName string) *storage.ComplianceOperatorScanConfigurationV2 {
	return &storage.ComplianceOperatorScanConfigurationV2{
		ScanName:               scanName,
		AutoApplyRemediations:  false,
		AutoUpdateRemediations: false,
		OneTimeScan:            false,
		Profiles: []*storage.ProfileShim{
			{
				ProfileId: "ocp4-cis",
			},
		},
		StrictNodeScan: false,
	}
}
