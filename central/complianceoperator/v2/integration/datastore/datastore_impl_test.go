//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/complianceoperator/v2/integration/store/postgres"
	apiV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestComplianceIntegrationDataStore(t *testing.T) {
	suite.Run(t, new(complianceIntegrationDataStoreTestSuite))
}

type complianceIntegrationDataStoreTestSuite struct {
	suite.Suite

	hasReadCtx   context.Context
	hasWriteCtx  context.Context
	noAccessCtx  context.Context
	testContexts map[string]context.Context

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
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
	s.noAccessCtx = sac.WithNoAccess(context.Background())
	s.testContexts = testutils.GetNamespaceScopedTestContexts(context.Background(), s.T(), resources.Compliance)

	s.db = pgtest.ForT(s.T())
	s.storage = postgres.New(s.db.DB)
	s.dataStore = New(s.storage, s.db.DB)
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
	testIntegrations[2].ComplianceNamespace = fixtureconsts.Namespace2
	s.NoError(s.dataStore.UpdateComplianceIntegration(s.hasWriteCtx, testIntegrations[2]))
	updated, exists, err := s.storage.Get(s.hasReadCtx, testIntegrations[2].GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(testIntegrations[2].GetId(), updated.GetId())
	s.Equal(fixtureconsts.Namespace2, updated.GetComplianceNamespace())

	// Now update integration index 1 to have same cluster/namespace as index 0.
	// Update should fail due to unique constraint on cluster
	testIntegrations[1].ClusterId = testIntegrations[0].GetClusterId()
	testIntegrations[1].ComplianceNamespace = testIntegrations[0].GetComplianceNamespace()
	s.Error(s.dataStore.UpdateComplianceIntegration(s.hasWriteCtx, testIntegrations[1]))
}

func (s *complianceIntegrationDataStoreTestSuite) TestGetComplianceIntegration() {
	testIntegrations := getDefaultTestIntegrations()
	ids := s.addBaseIntegrations(testIntegrations)
	for i, id := range ids {
		testIntegrations[i].Id = id
	}

	testCases := []struct {
		desc           string
		requestID      string
		scopeKey       string
		expectedID     string
		expectedResult *storage.ComplianceIntegration
	}{
		{
			desc:           "Existing integration - Full access",
			requestID:      ids[0],
			scopeKey:       testutils.UnrestrictedReadCtx,
			expectedID:     ids[0],
			expectedResult: testIntegrations[0],
		},
		{
			desc:           "Existing cluster 1 integration - Only cluster 1 access",
			requestID:      ids[0],
			scopeKey:       testutils.Cluster1ReadWriteCtx,
			expectedID:     ids[0],
			expectedResult: testIntegrations[0],
		},
		{
			desc:           "Existing cluster 1 integration - Only cluster 2 access",
			requestID:      ids[0],
			scopeKey:       testutils.Cluster2ReadWriteCtx,
			expectedID:     "",
			expectedResult: nil,
		},
	}
	for _, tc := range testCases {
		retrievedIntegration, exists, err := s.dataStore.GetComplianceIntegration(s.testContexts[tc.scopeKey], tc.requestID)
		s.NoError(err)
		s.True(exists != (tc.expectedResult == nil))
		protoassert.Equal(s.T(), tc.expectedResult, retrievedIntegration)
	}
}

func (s *complianceIntegrationDataStoreTestSuite) TestGetComplianceIntegrationByCluster() {
	testIntegrations := getDefaultTestIntegrations()
	ids := s.addBaseIntegrations(testIntegrations)
	for i, id := range ids {
		testIntegrations[i].Id = id
	}

	testCases := []struct {
		desc           string
		requestID      string
		scopeKey       string
		expectedID     string
		expectedResult *storage.ComplianceIntegration
	}{
		{
			desc:           "Existing Cluster 1 integration - Full access",
			requestID:      testconsts.Cluster1,
			scopeKey:       testutils.UnrestrictedReadCtx,
			expectedID:     ids[0],
			expectedResult: testIntegrations[0],
		},
		{
			desc:           "Existing cluster 1 integration - Only cluster 1 access",
			requestID:      testconsts.Cluster1,
			scopeKey:       testutils.Cluster1ReadWriteCtx,
			expectedID:     ids[0],
			expectedResult: testIntegrations[0],
		},
		{
			desc:           "Existing cluster 1 integration - Only cluster 2 access",
			requestID:      testconsts.Cluster1,
			scopeKey:       testutils.Cluster2ReadWriteCtx,
			expectedID:     "",
			expectedResult: nil,
		},
		{
			desc:           "Non existing cluster integration - Full access",
			requestID:      fixtureconsts.ClusterFake2,
			scopeKey:       testutils.UnrestrictedReadCtx,
			expectedID:     "",
			expectedResult: nil,
		},
	}
	for _, tc := range testCases {
		clusterIntegrations, err := s.dataStore.GetComplianceIntegrationByCluster(s.testContexts[tc.scopeKey], tc.requestID)
		s.NoError(err)
		// Set the ID to the result object if a result is expected.
		if tc.expectedResult != nil {
			protoassert.SliceContains(s.T(), clusterIntegrations, tc.expectedResult)
		} else {
			s.Empty(clusterIntegrations)
		}
	}
}

func (s *complianceIntegrationDataStoreTestSuite) TestGetComplianceIntegrations() {
	testIntegrations := getDefaultTestIntegrations()
	ids := s.addBaseIntegrations(testIntegrations)
	for i, id := range ids {
		testIntegrations[i].Id = id
	}

	testCases := []struct {
		desc           string
		query          *apiV1.Query
		scopeKey       string
		expectedID     []string
		expectedResult []*storage.ComplianceIntegration
	}{
		{
			desc:           "Empty Query - Full access",
			query:          search.NewQueryBuilder().ProtoQuery(),
			scopeKey:       testutils.UnrestrictedReadCtx,
			expectedID:     ids,
			expectedResult: testIntegrations,
		},
		{
			desc:           "Empty query - Only cluster 1 access",
			query:          search.NewQueryBuilder().ProtoQuery(),
			scopeKey:       testutils.Cluster1ReadWriteCtx,
			expectedID:     []string{ids[0]},
			expectedResult: []*storage.ComplianceIntegration{testIntegrations[0]},
		},
		{
			desc:           "Cluster 2 query - Only cluster 2 access",
			query:          search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:       testutils.Cluster2ReadWriteCtx,
			expectedID:     []string{ids[1]},
			expectedResult: []*storage.ComplianceIntegration{testIntegrations[1]},
		},
		{
			desc:           "Cluster 2 query - Only cluster 1 access",
			query:          search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:       testutils.Cluster1ReadWriteCtx,
			expectedID:     nil,
			expectedResult: nil,
		},
	}
	for _, tc := range testCases {
		clusterIntegrations, err := s.dataStore.GetComplianceIntegrations(s.testContexts[tc.scopeKey], tc.query)
		s.NoError(err)
		protoassert.SlicesEqual(s.T(), tc.expectedResult, clusterIntegrations)
	}
}

func (s *complianceIntegrationDataStoreTestSuite) TestGetComplianceIntegrationsView() {
	testIntegrations := getDefaultTestIntegrations()
	viewIntegrations := getDefaultTestIntegrationViews()
	ids := s.addBaseIntegrations(testIntegrations)
	for i, id := range ids {
		testIntegrations[i].Id = id
		viewIntegrations[i].ID = id
	}

	// Add some clusters
	_, err := s.db.DB.Exec(context.Background(), "insert into clusters (id, name, status_providermetadata_cluster_type, type) values ($1, $2, $3, $4)", testconsts.Cluster1, "cluster1", 1, 1)
	s.Require().NoError(err)
	_, err = s.db.DB.Exec(context.Background(), "insert into clusters (id, name, status_providermetadata_cluster_type, type) values ($1, $2, $3, $4)", testconsts.Cluster2, "cluster2", 2, 2)
	s.Require().NoError(err)
	_, err = s.db.DB.Exec(context.Background(), "insert into clusters (id, name, status_providermetadata_cluster_type, type) values ($1, $2, $3, $4)", testconsts.Cluster3, "cluster3", 5, 5)
	s.Require().NoError(err)

	testCases := []struct {
		desc           string
		query          *apiV1.Query
		scopeKey       string
		expectedID     []string
		expectedResult []*IntegrationDetails
	}{
		{
			desc:           "Empty Query - Full access",
			query:          search.NewQueryBuilder().ProtoQuery(),
			scopeKey:       testutils.UnrestrictedReadCtx,
			expectedID:     ids,
			expectedResult: viewIntegrations,
		},
		{
			desc:           "Empty query - Only cluster 1 access",
			query:          search.NewQueryBuilder().ProtoQuery(),
			scopeKey:       testutils.Cluster1ReadWriteCtx,
			expectedID:     []string{ids[0]},
			expectedResult: []*IntegrationDetails{viewIntegrations[0]},
		},
		{
			desc:           "Cluster 2 query - Only cluster 2 access",
			query:          search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:       testutils.Cluster2ReadWriteCtx,
			expectedID:     []string{ids[1]},
			expectedResult: []*IntegrationDetails{viewIntegrations[1]},
		},
		{
			desc:           "Cluster 2 query - Only cluster 1 access",
			query:          search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:       testutils.Cluster1ReadWriteCtx,
			expectedID:     nil,
			expectedResult: nil,
		},
	}
	for _, tc := range testCases {
		clusterIntegrations, err := s.dataStore.GetComplianceIntegrationsView(s.testContexts[tc.scopeKey], tc.query)
		s.Require().NoError(err)
		// style is killing me because the struct references an enum in a proto so it wants to use protoassert but
		// since the struct is not a proto, the protoassert doesn't work.
		for idx, integration := range clusterIntegrations {
			assert.Equal(s.T(), tc.expectedResult[idx].ID, integration.ID)
			assert.Equal(s.T(), tc.expectedResult[idx].ClusterID, integration.ClusterID)
			assert.Equal(s.T(), tc.expectedResult[idx].ClusterName, integration.ClusterName)
			assert.Equal(s.T(), tc.expectedResult[idx].Version, integration.Version)
			assert.Equal(s.T(), tc.expectedResult[idx].OperatorStatus.String(), integration.OperatorStatus.String())
			assert.Equal(s.T(), tc.expectedResult[idx].OperatorInstalled, integration.OperatorInstalled)
			assert.Equal(s.T(), tc.expectedResult[idx].StatusProviderMetadataClusterType.String(), integration.StatusProviderMetadataClusterType.String())
			assert.Equal(s.T(), tc.expectedResult[idx].Type.String(), integration.Type.String())
		}
	}
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
	protoassert.SliceNotContains(s.T(), integrations, testIntegrations[0])
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
	err = s.dataStore.RemoveComplianceIntegrationByCluster(s.hasWriteCtx, testconsts.Cluster1)
	s.NoError(err)
	integrations, _, err = s.storage.GetMany(s.hasWriteCtx, ids)
	s.NoError(err)
	s.Equal(len(ids)-1, len(integrations))
	protoassert.SliceNotContains(s.T(), integrations, testIntegrations[0])
	protoassert.SliceContains(s.T(), integrations, testIntegrations[1])
	protoassert.SliceContains(s.T(), integrations, testIntegrations[2])
}

func (s *complianceIntegrationDataStoreTestSuite) TestCountIntegrations() {
	testIntegrations := getDefaultTestIntegrations()
	ids := s.addBaseIntegrations(testIntegrations)
	for i, id := range ids {
		testIntegrations[i].Id = id
	}

	testCases := []struct {
		desc           string
		query          *apiV1.Query
		scopeKey       string
		expectedResult int
	}{
		{
			desc:           "Empty Query - Full access",
			query:          search.NewQueryBuilder().ProtoQuery(),
			scopeKey:       testutils.UnrestrictedReadCtx,
			expectedResult: len(testIntegrations),
		},
		{
			desc:           "Empty query - Only cluster 1 access",
			query:          search.NewQueryBuilder().ProtoQuery(),
			scopeKey:       testutils.Cluster1ReadWriteCtx,
			expectedResult: 1,
		},
		{
			desc:           "Cluster 2 query - Only cluster 2 access",
			query:          search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:       testutils.Cluster2ReadWriteCtx,
			expectedResult: 1,
		},
		{
			desc:           "Cluster 2 query - Only cluster 1 access",
			query:          search.NewQueryBuilder().AddStrings(search.ClusterID, testconsts.Cluster2).ProtoQuery(),
			scopeKey:       testutils.Cluster1ReadWriteCtx,
			expectedResult: 0,
		},
	}
	for _, tc := range testCases {
		count, err := s.dataStore.CountIntegrations(s.testContexts[tc.scopeKey], tc.query)
		s.NoError(err)
		s.Equal(tc.expectedResult, count)
	}
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
			Id:                  "",
			ClusterId:           testconsts.Cluster1,
			ComplianceNamespace: fixtureconsts.Namespace1,
			Version:             "2",
			OperatorInstalled:   true,
			OperatorStatus:      storage.COStatus_HEALTHY,
		},
		{
			Id:                  "",
			ClusterId:           testconsts.Cluster2,
			ComplianceNamespace: fixtureconsts.Namespace1,
			Version:             "2",
			OperatorInstalled:   true,
		},
		{
			Id:                  "",
			ClusterId:           testconsts.Cluster3,
			ComplianceNamespace: fixtureconsts.Namespace1,
			Version:             "2",
			OperatorInstalled:   true,
		},
	}

	return integrations
}

func getDefaultTestIntegrationViews() []*IntegrationDetails {
	integrations := []*IntegrationDetails{
		{
			ID:                                "",
			Version:                           "2",
			OperatorInstalled:                 pointers.Bool(true),
			OperatorStatus:                    pointers.Pointer(storage.COStatus_HEALTHY),
			ClusterID:                         testconsts.Cluster1,
			ClusterName:                       "cluster1",
			Type:                              pointers.Pointer(storage.ClusterType_KUBERNETES_CLUSTER),
			StatusProviderMetadataClusterType: pointers.Pointer(storage.ClusterMetadata_AKS),
		},
		{
			ID:                                "",
			ClusterID:                         testconsts.Cluster2,
			Version:                           "2",
			OperatorStatus:                    pointers.Pointer(storage.COStatus_HEALTHY),
			ClusterName:                       "cluster2",
			Type:                              pointers.Pointer(storage.ClusterType_OPENSHIFT_CLUSTER),
			StatusProviderMetadataClusterType: pointers.Pointer(storage.ClusterMetadata_ARO),
			OperatorInstalled:                 pointers.Bool(true),
		},
		{
			ID:                                "",
			ClusterID:                         testconsts.Cluster3,
			Version:                           "2",
			OperatorStatus:                    pointers.Pointer(storage.COStatus_HEALTHY),
			ClusterName:                       "cluster3",
			Type:                              pointers.Pointer(storage.ClusterType_OPENSHIFT4_CLUSTER),
			StatusProviderMetadataClusterType: pointers.Pointer(storage.ClusterMetadata_OCP),
			OperatorInstalled:                 pointers.Bool(true),
		},
	}

	return integrations
}
