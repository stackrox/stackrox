//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	clusterMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
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

	hasReadCtx  context.Context
	hasWriteCtx context.Context
	noAccessCtx context.Context

	dataStore        DataStore
	db               *pgtest.TestPostgres
	storage          configStore.Store
	statusStorage    scanStatusStore.Store
	clusterDatastore *clusterMocks.MockDataStore
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
			sac.ResourceScopeKeys(resources.ComplianceOperator)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ComplianceOperator)))
	s.noAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())

	s.mockCtrl = gomock.NewController(s.T())
	s.clusterDatastore = clusterMocks.NewMockDataStore(s.mockCtrl)

	s.db = pgtest.ForT(s.T())
	var err error
	s.storage = configStore.New(s.db)
	s.statusStorage = scanStatusStore.New(s.db)
	s.Require().NoError(err)

	s.dataStore = New(s.storage, s.statusStorage, s.clusterDatastore)
}

func (s *complianceScanConfigDataStoreTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *complianceScanConfigDataStoreTestSuite) TestGetScanConfiguration() {
	configID := uuid.NewV4().String()

	scanConfig := getTestRec()
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

	scanConfig := getTestRec()
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

func (s *complianceScanConfigDataStoreTestSuite) TestScanConfigurationExists() {
	configID := uuid.NewV4().String()

	scanConfig := getTestRec()
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

	scanConfig := getTestRec()
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

	scanConfig := getTestRec()
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
	configID := uuid.NewV4().String()

	scanConfig := getTestRec()
	scanConfig.Id = configID

	s.clusterDatastore.EXPECT().GetCluster(gomock.Any(), fixtureconsts.Cluster1).Return(&storage.Cluster{
		Id:   fixtureconsts.Cluster1,
		Name: mockClusterName,
	}, true, nil).Times(2)

	// Add a record so we have something to find
	s.Require().NoError(s.storage.Upsert(s.hasWriteCtx, scanConfig))

	// Add Scan config status
	s.Require().NoError(s.dataStore.UpdateClusterStatus(s.hasWriteCtx, configID, fixtureconsts.Cluster1, "testing status"))

	clusterStatuses, err := s.dataStore.GetScanConfigClusterStatus(s.hasReadCtx, configID)
	s.Require().NoError(err)
	s.Require().Equal(1, len(clusterStatuses))

	// Try to add one with no existing scan config
	s.Require().NotNil(s.dataStore.UpdateClusterStatus(s.hasWriteCtx, uuid.NewDummy().String(), fixtureconsts.Cluster1, "testing status"))
	clusterStatuses, err = s.dataStore.GetScanConfigClusterStatus(s.hasReadCtx, uuid.NewDummy().String())
	s.Require().NoError(err)
	s.Require().Equal(0, len(clusterStatuses))

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
