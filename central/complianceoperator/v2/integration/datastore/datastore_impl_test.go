//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/complianceoperator/v2/integration/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestComplianceIntegrationDataStore(t *testing.T) {
	suite.Run(t, new(complianceIntegrationDataStoreTestSuite))
}

type complianceIntegrationDataStoreTestSuite struct {
	suite.Suite

	hasReadCtx  context.Context
	hasWriteCtx context.Context
	noAccessCtx context.Context

	dataStore DataStore
	db        *pgtest.TestPostgres
	storage   postgres.Store
}

func (s *complianceIntegrationDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")
	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skip("Skip tests when ComplianceEnhancements disabled")
		s.T().SkipNow()
	}
}

func (s *complianceIntegrationDataStoreTestSuite) SetupTest() {
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	s.noAccessCtx = sac.WithNoAccess(context.Background())

	s.db = pgtest.ForT(s.T())
	var err error
	s.storage = postgres.New(s.db)
	s.Require().NoError(err)

	s.dataStore = New(s.storage)
}

func (s *complianceIntegrationDataStoreTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *complianceIntegrationDataStoreTestSuite) TestAddComplianceIntegration() {
	testIntegrations := getDefaultTestIntegrations()
	// This will add entries by calling AddComplianceIntegration
	_ = s.addBaseIntegrations(testIntegrations)

	// Try to re-add an integration.  Should return an error.
	_, err := s.dataStore.AddComplianceIntegration(s.hasWriteCtx, testIntegrations[0])
	s.Error(err)
}

func (s *complianceIntegrationDataStoreTestSuite) TestUpdateComplianceIntegration() {
	testIntegrations := getDefaultTestIntegrations()
	_ = s.addBaseIntegrations(testIntegrations)

	// Update namespace and update
	testIntegrations[2].NamespaceId = fixtureconsts.Namespace2
	s.NoError(s.dataStore.UpdateComplianceIntegration(s.hasWriteCtx, testIntegrations[2]))
	updated, exists, err := s.storage.Get(s.hasReadCtx, testIntegrations[2].GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(testIntegrations[2].GetId(), updated.GetId())
	s.Equal(fixtureconsts.Namespace2, updated.GetNamespaceId())

	// Now update integration index 1 to have same cluster/namespace as index 0.
	// Update should fail due to unique constraint on cluster
	testIntegrations[1].ClusterId = testIntegrations[0].GetClusterId()
	testIntegrations[1].Namespace = testIntegrations[0].GetNamespace()
	testIntegrations[1].NamespaceId = testIntegrations[0].GetNamespaceId()
	s.Error(s.dataStore.UpdateComplianceIntegration(s.hasWriteCtx, testIntegrations[1]))
}

func (s *complianceIntegrationDataStoreTestSuite) TestGetComplianceIntegration() {
	testIntegrations := getDefaultTestIntegrations()
	ids := s.addBaseIntegrations(testIntegrations)

	// Exists
	retrievedIntegration, exists, err := s.dataStore.GetComplianceIntegration(s.hasReadCtx, ids[0])
	s.NoError(err)
	s.True(exists)
	s.Equal(testIntegrations[0], retrievedIntegration)

	// Does not exist
	retrievedIntegration, exists, err = s.dataStore.GetComplianceIntegration(s.hasReadCtx, uuid.NewV4().String())
	s.NoError(err)
	s.False(exists)
	s.Nil(retrievedIntegration)
}

func (s *complianceIntegrationDataStoreTestSuite) TestGetComplianceIntegrations() {
	testIntegrations := getDefaultTestIntegrations()
	_ = s.addBaseIntegrations(testIntegrations)

	// No integrations found for cluster
	clusterIntegrations, err := s.dataStore.GetComplianceIntegrations(s.hasReadCtx, uuid.NewV4().String())
	s.NoError(err)
	s.Nil(clusterIntegrations)

	// Use cluster 1
	clusterIntegrations, err = s.dataStore.GetComplianceIntegrations(s.hasReadCtx, fixtureconsts.Cluster1)
	s.NoError(err)
	s.Equal(1, len(clusterIntegrations))
	s.Contains(clusterIntegrations, testIntegrations[0])
}

func (s *complianceIntegrationDataStoreTestSuite) TestRemoveComplianceIntegration() {
	testIntegrations := getDefaultTestIntegrations()
	ids := s.addBaseIntegrations(testIntegrations)

	// Try to remove non-existent id
	err := s.dataStore.RemoveComplianceIntegration(s.hasWriteCtx, uuid.NewV4().String())
	s.NoError(err)
	integrations, _, err := s.storage.GetMany(s.hasWriteCtx, ids)
	s.NoError(err)
	s.Equal(len(ids), len(integrations))

	// Remove one
	err = s.dataStore.RemoveComplianceIntegration(s.hasWriteCtx, ids[0])
	s.NoError(err)
	integrations, _, err = s.storage.GetMany(s.hasWriteCtx, ids)
	s.NoError(err)
	s.Greater(len(ids), len(integrations))
	s.NotContains(integrations, testIntegrations[0])
}

func (s *complianceIntegrationDataStoreTestSuite) TestRemoveComplianceIntegrationByCluster() {
	testIntegrations := getDefaultTestIntegrations()
	ids := s.addBaseIntegrations(testIntegrations)

	// Try to remove non-existent id
	err := s.dataStore.RemoveComplianceIntegrationByCluster(s.hasWriteCtx, uuid.NewV4().String())
	s.NoError(err)
	integrations, _, err := s.storage.GetMany(s.hasWriteCtx, ids)
	s.NoError(err)
	s.Equal(len(ids), len(integrations))

	// Remove integrations with cluster 1
	err = s.dataStore.RemoveComplianceIntegrationByCluster(s.hasWriteCtx, fixtureconsts.Cluster1)
	s.NoError(err)
	integrations, _, err = s.storage.GetMany(s.hasWriteCtx, ids)
	s.NoError(err)
	s.Equal(len(ids)-1, len(integrations))
	s.NotContains(integrations, testIntegrations[0])
	s.Contains(integrations, testIntegrations[1])
	s.Contains(integrations, testIntegrations[2])
}

func (s *complianceIntegrationDataStoreTestSuite) addBaseIntegrations(testIntegrations []*storage.ComplianceIntegration) []string {
	var ids []string
	for _, integration := range testIntegrations {
		id, err := s.dataStore.AddComplianceIntegration(s.hasWriteCtx, integration)
		s.NoError(err)
		s.NotEmpty(id)
		integration.Id = id
		ids = append(ids, id)
	}

	return ids
}

func getDefaultTestIntegrations() []*storage.ComplianceIntegration {
	integrations := []*storage.ComplianceIntegration{
		{
			Id:          "",
			ClusterId:   fixtureconsts.Cluster1,
			NamespaceId: fixtureconsts.Namespace1,
			Version:     "2",
		},
		{
			Id:          "",
			ClusterId:   fixtureconsts.Cluster2,
			NamespaceId: fixtureconsts.Namespace1,
			Version:     "2",
		},
		{
			Id:          "",
			ClusterId:   fixtureconsts.Cluster3,
			NamespaceId: fixtureconsts.Namespace1,
			Version:     "2",
		},
	}

	return integrations
}
