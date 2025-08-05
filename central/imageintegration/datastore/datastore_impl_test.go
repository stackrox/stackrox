//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/imageintegration/store"
	postgresStore "github.com/stackrox/rox/central/imageintegration/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	clusterID = "12841143-0b39-47be-8eca-5bca52b4a288"
)

func TestImageIntegrationDataStore(t *testing.T) {
	suite.Run(t, new(ImageIntegrationDataStoreTestSuite))
}

type ImageIntegrationDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	datastore DataStore

	testDB *pgtest.TestPostgres
	store  store.Store
}

func (suite *ImageIntegrationDataStoreTestSuite) SetupTest() {
	suite.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	suite.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))
	suite.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration)))

	suite.testDB = pgtest.ForT(suite.T())
	suite.NotNil(suite.testDB)

	suite.store = postgresStore.New(suite.testDB.DB)

	// test formattedSearcher
	suite.datastore = NewForTestOnly(suite.store)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestIntegrationsPersistence() {
	testIntegrations(suite.T(), suite.store, suite.datastore)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestIntegrations() {
	testIntegrations(suite.T(), suite.store, suite.datastore)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestIntegrationsFiltering() {
	integrations := []*storage.ImageIntegration{
		{
			Id:   uuid.NewV4().String(),
			Name: "registry1",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint1",
				},
			},
		},
		{
			Id:   uuid.NewV4().String(),
			Name: "registry2",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint2",
				},
			},
		},
	}
	// Test Add
	for _, r := range integrations {
		id, err := suite.datastore.AddImageIntegration(suite.hasWriteCtx, r)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	actualIntegrations, err := suite.datastore.GetImageIntegrations(suite.hasWriteCtx, &v1.GetImageIntegrationsRequest{})
	suite.NoError(err)
	protoassert.ElementsMatch(suite.T(), integrations, actualIntegrations)
}

func testIntegrations(t *testing.T, insertStorage store.Store, retrievalStorage DataStore) {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Integration)))
	integrations := []*storage.ImageIntegration{
		{
			Id:   uuid.NewV4().String(),
			Name: "registry1",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint1",
				},
			},
		},
		{
			Id:   uuid.NewV4().String(),
			Name: "registry2",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint2",
				},
			},
		},
	}

	// Test Add
	for _, r := range integrations {
		err := insertStorage.Upsert(ctx, r)
		assert.NoError(t, err)
	}
	for _, r := range integrations {
		got, exists, err := retrievalStorage.GetImageIntegration(ctx, r.GetId())
		assert.NoError(t, err)
		assert.True(t, exists)
		protoassert.Equal(t, got, r)
	}

	// Test Update
	for _, r := range integrations {
		r.Name += "/api"
	}

	for _, r := range integrations {
		assert.NoError(t, insertStorage.Upsert(ctx, r))
	}

	for _, r := range integrations {
		got, exists, err := retrievalStorage.GetImageIntegration(ctx, r.GetId())
		assert.NoError(t, err)
		assert.True(t, exists)
		protoassert.Equal(t, got, r)
	}

	// Test Remove
	for _, r := range integrations {
		assert.NoError(t, insertStorage.Delete(ctx, r.GetId()))
	}

	for _, r := range integrations {
		_, exists, err := retrievalStorage.GetImageIntegration(ctx, r.GetId())
		assert.NoError(t, err)
		assert.False(t, exists)
	}
}

func getIntegration(name string) *storage.ImageIntegration {
	return &storage.ImageIntegration{
		Id:   uuid.NewV4().String(),
		Name: name,
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "https://endpoint1",
			},
		},
	}
}

func (suite *ImageIntegrationDataStoreTestSuite) storeIntegration(name string) *storage.ImageIntegration {
	integration := getIntegration(name)
	err := suite.store.Upsert(suite.hasWriteCtx, integration)
	suite.NoError(err)
	return integration
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesGet() {
	group, exists, err := suite.datastore.GetImageIntegration(suite.hasNoneCtx, "Some ID")
	suite.NoError(err, "expected no error, should return nil without access")
	suite.False(exists, "expected exists to be false as access was denied and bools can't be nil")
	suite.Nil(group, "expected return value to be nil")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsGet() {
	integration := suite.storeIntegration("Joseph Rules")

	gotInt, exists, err := suite.datastore.GetImageIntegration(suite.hasReadCtx, integration.GetId())
	suite.NoError(err, "expected no error trying to read with permissions")
	protoassert.Equal(suite.T(), integration, gotInt)
	suite.True(exists)

	gotInt, exists, err = suite.datastore.GetImageIntegration(suite.hasWriteCtx, integration.GetId())
	suite.NoError(err, "expected no error trying to read with permissions")
	protoassert.Equal(suite.T(), integration, gotInt)
	suite.True(exists)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesGetBatch() {
	integrations, err := suite.datastore.GetImageIntegrations(suite.hasNoneCtx, &v1.GetImageIntegrationsRequest{})
	suite.NoError(err, "expected no error, should return nil without access")
	suite.Nil(integrations, "expected return value to be nil")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsGetBatch() {
	integration := suite.storeIntegration("Some Integration")
	integrationList := []*storage.ImageIntegration{integration}

	getRequest := &v1.GetImageIntegrationsRequest{Name: integration.GetName(), Cluster: integration.GetClusterId()}

	gotImages, err := suite.datastore.GetImageIntegrations(suite.hasReadCtx, getRequest)
	suite.NoError(err, "expected no error trying to read with permissions")
	protoassert.ElementsMatch(suite.T(), integrationList, gotImages)

	gotImages, err = suite.datastore.GetImageIntegrations(suite.hasWriteCtx, getRequest)
	suite.NoError(err, "expected no error trying to read with permissions")
	protoassert.ElementsMatch(suite.T(), integrationList, gotImages)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesAdd() {
	integrationOne := getIntegration("some kinda name")
	id, err := suite.datastore.AddImageIntegration(suite.hasNoneCtx, integrationOne)
	suite.Error(err, "expected an error trying to write without permissions")
	suite.Empty(id)

	integrationTwo := getIntegration("Get named, you")
	id, err = suite.datastore.AddImageIntegration(suite.hasReadCtx, integrationTwo)
	suite.Error(err, "expected an error trying to write without permissions")
	suite.Empty(id)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsAdd() {
	id, err := suite.datastore.AddImageIntegration(suite.hasWriteCtx, getIntegration("namenamenamename"))
	suite.NoError(err, "expected no error trying to write with permissions")
	suite.NotEmpty(id)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesUpdate() {
	integration := suite.storeIntegration("name")

	err := suite.datastore.UpdateImageIntegration(suite.hasNoneCtx, integration)
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.UpdateImageIntegration(suite.hasReadCtx, integration)
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsUpdate() {
	integration := suite.storeIntegration("joseph is the best")

	err := suite.datastore.UpdateImageIntegration(suite.hasWriteCtx, integration)
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestEnforcesRemove() {
	err := suite.datastore.RemoveImageIntegration(suite.hasNoneCtx, "blerk")
	suite.Error(err, "expected an error trying to write without permissions")

	err = suite.datastore.RemoveImageIntegration(suite.hasReadCtx, "hkddsfk")
	suite.Error(err, "expected an error trying to write without permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestAllowsRemove() {
	integration := suite.storeIntegration("jdgbfdkjh")

	err := suite.datastore.RemoveImageIntegration(suite.hasWriteCtx, integration.GetId())
	suite.NoError(err, "expected no error trying to write with permissions")
}

func (suite *ImageIntegrationDataStoreTestSuite) TestSearch() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	_, err := suite.datastore.Search(ctx, nil)
	suite.NoError(err)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestIndexing() {
	ii := &storage.ImageIntegration{
		Id:        uuid.NewV4().String(),
		ClusterId: clusterID,
		Name:      "imageIntegration1",
	}

	suite.NoError(suite.store.Upsert(sac.WithAllAccess(context.Background()), ii))

	q := pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, clusterID).ProtoQuery()
	results, err := suite.store.Search(suite.hasWriteCtx, q)
	suite.NoError(err)
	suite.Len(results, 1)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestDataStoreSearch() {
	ii := &storage.ImageIntegration{
		Id:        "id1",
		ClusterId: clusterID,
		Name:      "imageIntegration1",
	}

	// Create a new datastore since the one in suite uses mocks
	ds := New(suite.store)

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())
	_, err := ds.AddImageIntegration(ctx, ii)
	suite.NoError(err)

	q := pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, clusterID).ProtoQuery()
	results, err := ds.Search(ctx, q)
	suite.NoError(err)
	suite.Len(results, 1)
}

func (suite *ImageIntegrationDataStoreTestSuite) TestSearchWithMultipleIntegrations() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	// Generate test cluster IDs as proper UUIDs
	cluster1ID := uuid.NewV4().String()
	cluster2ID := uuid.NewV4().String()
	cluster3ID := uuid.NewV4().String()

	// Create multiple test image integrations with different properties
	integrations := []*storage.ImageIntegration{
		{
			Id:        uuid.NewV4().String(),
			Name:      "docker-registry-1",
			ClusterId: cluster1ID,
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://docker-registry-1.com",
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "docker-registry-2",
			ClusterId: cluster2ID,
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://docker-registry-2.com",
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "quay-registry",
			ClusterId: cluster1ID,
			IntegrationConfig: &storage.ImageIntegration_Quay{
				Quay: &storage.QuayConfig{
					Endpoint: "https://quay.io",
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "gcr-registry",
			ClusterId: cluster3ID,
			IntegrationConfig: &storage.ImageIntegration_Google{
				Google: &storage.GoogleConfig{
					Endpoint: "https://gcr.io",
				},
			},
		},
	}

	// Add all integrations
	for _, integration := range integrations {
		_, err := suite.datastore.AddImageIntegration(ctx, integration)
		suite.NoError(err)
	}

	testCases := []struct {
		name          string
		query         *v1.Query
		expectedCount int
		description   string
	}{
		{
			name:          "Empty query - should return all integrations",
			query:         pkgSearch.EmptyQuery(),
			expectedCount: 4,
			description:   "Empty query should return all image integrations",
		},
		{
			name:          "Search by cluster ID - cluster-1",
			query:         pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, cluster1ID).ProtoQuery(),
			expectedCount: 2,
			description:   "Should return integrations from cluster-1",
		},
		{
			name:          "Search by cluster ID - cluster-2",
			query:         pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, cluster2ID).ProtoQuery(),
			expectedCount: 1,
			description:   "Should return integrations from cluster-2",
		},
		{
			name:          "Search by cluster ID - cluster-3",
			query:         pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, cluster3ID).ProtoQuery(),
			expectedCount: 1,
			description:   "Should return integrations from cluster-3",
		},
		{
			name:          "Search by cluster ID - non-existent cluster",
			query:         pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, uuid.NewV4().String()).ProtoQuery(),
			expectedCount: 0,
			description:   "Should return no integrations for non-existent cluster",
		},
		{
			name:          "Search by ID - exact match",
			query:         pkgSearch.NewQueryBuilder().AddDocIDs(integrations[0].GetId()).ProtoQuery(),
			expectedCount: 1,
			description:   "Should return exact ID match",
		},
		{
			name:          "Search by ID - non-existent ID",
			query:         pkgSearch.NewQueryBuilder().AddDocIDs(uuid.NewV4().String()).ProtoQuery(),
			expectedCount: 0,
			description:   "Should return no integrations for non-existent ID",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			results, err := suite.datastore.Search(ctx, tc.query)
			suite.NoError(err, tc.description)
			suite.Len(results, tc.expectedCount, tc.description)

			// Create maps for easier verification
			integrationByID := make(map[string]*storage.ImageIntegration)
			integrationsByCluster := make(map[string][]*storage.ImageIntegration)
			for _, integration := range integrations {
				integrationByID[integration.GetId()] = integration
				integrationsByCluster[integration.GetClusterId()] = append(integrationsByCluster[integration.GetClusterId()], integration)
			}

			// Verify specific test case expectations
			switch tc.name {
			case "Empty query - should return all integrations":
				// Should return all 4 integrations
				returnedIDs := make(map[string]bool)
				for _, result := range results {
					returnedIDs[result.ID] = true
					suite.True(integrationByID[result.ID] != nil, "Returned ID should be from our test integrations")
				}
				suite.Len(returnedIDs, 4, "Should return all 4 unique integration IDs")

			case "Search by cluster ID - cluster-1":
				// Should return integrations from cluster1ID
				for _, result := range results {
					integration := integrationByID[result.ID]
					suite.NotNil(integration, "Integration should exist")
					suite.Equal(cluster1ID, integration.GetClusterId(), "Integration should be from cluster-1")
				}

			case "Search by cluster ID - cluster-2":
				// Should return integrations from cluster2ID
				for _, result := range results {
					integration := integrationByID[result.ID]
					suite.NotNil(integration, "Integration should exist")
					suite.Equal(cluster2ID, integration.GetClusterId(), "Integration should be from cluster-2")
				}

			case "Search by cluster ID - cluster-3":
				// Should return integrations from cluster3ID
				for _, result := range results {
					integration := integrationByID[result.ID]
					suite.NotNil(integration, "Integration should exist")
					suite.Equal(cluster3ID, integration.GetClusterId(), "Integration should be from cluster-3")
				}

			case "Search by ID - exact match":
				// Should return exactly the first integration
				suite.Len(results, 1, "Should return exactly one result")
				suite.Equal(integrations[0].GetId(), results[0].ID, "Should return the correct integration ID")

			case "Search by cluster ID - non-existent cluster", "Search by ID - non-existent ID":
				// Should return empty results - already verified by length check

			default:
				// For any other test cases, just verify all returned IDs exist in our test data
				for _, result := range results {
					suite.True(integrationByID[result.ID] != nil, "Returned ID should be from our test integrations")
				}
			}
		})
	}
}

func (suite *ImageIntegrationDataStoreTestSuite) TestCountWithMultipleIntegrations() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	// Generate test cluster IDs as proper UUIDs
	countCluster1ID := uuid.NewV4().String()
	countCluster2ID := uuid.NewV4().String()

	// Create test integrations with different properties for count testing
	integrations := []*storage.ImageIntegration{
		{
			Id:        uuid.NewV4().String(),
			Name:      "integration-count-1",
			ClusterId: countCluster1ID,
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://count-test-1.com",
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "integration-count-2",
			ClusterId: countCluster1ID,
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://count-test-2.com",
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "integration-count-3",
			ClusterId: countCluster2ID,
			IntegrationConfig: &storage.ImageIntegration_Quay{
				Quay: &storage.QuayConfig{
					Endpoint: "https://quay-count-test.io",
				},
			},
		},
	}

	// Add all integrations
	for _, integration := range integrations {
		_, err := suite.datastore.AddImageIntegration(ctx, integration)
		suite.NoError(err)
	}

	testCases := []struct {
		name          string
		query         *v1.Query
		expectedCount int
		description   string
	}{
		{
			name:          "Count all integrations - empty query",
			query:         pkgSearch.EmptyQuery(),
			expectedCount: 3,
			description:   "Should count all integrations when no filter applied",
		},
		{
			name:          "Count by cluster ID - count-cluster-1",
			query:         pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, countCluster1ID).ProtoQuery(),
			expectedCount: 2,
			description:   "Should count integrations in count-cluster-1",
		},
		{
			name:          "Count by cluster ID - count-cluster-2",
			query:         pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, countCluster2ID).ProtoQuery(),
			expectedCount: 1,
			description:   "Should count integrations in count-cluster-2",
		},
		{
			name:          "Count by cluster ID - non-existent cluster",
			query:         pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, uuid.NewV4().String()).ProtoQuery(),
			expectedCount: 0,
			description:   "Should return zero count for non-existent cluster",
		},
		{
			name:          "Count by ID - specific integration",
			query:         pkgSearch.NewQueryBuilder().AddDocIDs(integrations[0].GetId()).ProtoQuery(),
			expectedCount: 1,
			description:   "Should count specific integration by ID",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			count, err := suite.datastore.Count(ctx, tc.query)
			suite.NoError(err, tc.description)
			suite.Equal(tc.expectedCount, count, tc.description)
		})
	}
}

func (suite *ImageIntegrationDataStoreTestSuite) TestSearchImageIntegrationsWithMultiple() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	// Generate test cluster IDs as proper UUIDs
	searchCluster1ID := uuid.NewV4().String()
	searchCluster2ID := uuid.NewV4().String()

	// Create test integrations with different properties
	integrations := []*storage.ImageIntegration{
		{
			Id:        uuid.NewV4().String(),
			Name:      "search-integration-1",
			ClusterId: searchCluster1ID,
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://search-test-1.com",
					Username: "user1",
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "search-integration-2",
			ClusterId: searchCluster1ID,
			IntegrationConfig: &storage.ImageIntegration_Quay{
				Quay: &storage.QuayConfig{
					Endpoint: "https://quay-search-test.io",
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "search-gcr-integration",
			ClusterId: searchCluster2ID,
			IntegrationConfig: &storage.ImageIntegration_Google{
				Google: &storage.GoogleConfig{
					Endpoint: "https://gcr-search-test.io",
				},
			},
		},
	}

	// Add all integrations
	for _, integration := range integrations {
		_, err := suite.datastore.AddImageIntegration(ctx, integration)
		suite.NoError(err)
	}

	testCases := []struct {
		name          string
		query         *v1.Query
		expectedCount int
		description   string
	}{
		{
			name:          "SearchImageIntegrations - empty query",
			query:         pkgSearch.EmptyQuery(),
			expectedCount: 3,
			description:   "Should return all integrations as SearchResults",
		},
		{
			name:          "SearchImageIntegrations - by cluster ID",
			query:         pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, searchCluster1ID).ProtoQuery(),
			expectedCount: 2,
			description:   "Should return SearchResults for integrations in search-cluster-1",
		},
		{
			name:          "SearchImageIntegrations - by ID",
			query:         pkgSearch.NewQueryBuilder().AddDocIDs(integrations[2].GetId()).ProtoQuery(),
			expectedCount: 1,
			description:   "Should return SearchResult for specific integration ID",
		},
		{
			name:          "SearchImageIntegrations - no matches",
			query:         pkgSearch.NewQueryBuilder().AddDocIDs(uuid.NewV4().String()).ProtoQuery(),
			expectedCount: 0,
			description:   "Should return empty SearchResults for non-matching ID",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			searchResults, err := suite.datastore.SearchImageIntegrations(ctx, tc.query)
			suite.NoError(err, tc.description)
			suite.Len(searchResults, tc.expectedCount, tc.description)

			// Create map for easier verification
			integrationByID := make(map[string]*storage.ImageIntegration)
			for _, integration := range integrations {
				integrationByID[integration.GetId()] = integration
			}

			// Verify specific test case expectations
			switch tc.name {
			case "SearchImageIntegrations - empty query":
				// Should return all 3 integrations
				returnedIDs := make(map[string]bool)
				for _, searchResult := range searchResults {
					suite.Equal(v1.SearchCategory_IMAGE_INTEGRATIONS, searchResult.Category, "Category should be IMAGE_INTEGRATIONS")
					suite.NotEmpty(searchResult.Id, "SearchResult should have an ID")
					suite.NotEmpty(searchResult.Name, "SearchResult should have a name")

					integration := integrationByID[searchResult.Id]
					suite.NotNil(integration, "Integration should exist")
					suite.Equal(integration.GetName(), searchResult.Name, "SearchResult name should match integration name")
					returnedIDs[searchResult.Id] = true
				}
				suite.Len(returnedIDs, 3, "Should return all 3 unique integration IDs")

			case "SearchImageIntegrations - by cluster ID":
				// Should return integrations from searchCluster1ID
				for _, searchResult := range searchResults {
					suite.Equal(v1.SearchCategory_IMAGE_INTEGRATIONS, searchResult.Category, "Category should be IMAGE_INTEGRATIONS")
					suite.NotEmpty(searchResult.Id, "SearchResult should have an ID")
					suite.NotEmpty(searchResult.Name, "SearchResult should have a name")

					integration := integrationByID[searchResult.Id]
					suite.NotNil(integration, "Integration should exist")
					suite.Equal(integration.GetName(), searchResult.Name, "SearchResult name should match integration name")
					suite.Equal(searchCluster1ID, integration.GetClusterId(), "Integration should be from search-cluster-1")
				}

			case "SearchImageIntegrations - by ID":
				// Should return exactly the third integration (integrations[2])
				suite.Len(searchResults, 1, "Should return exactly one result")
				searchResult := searchResults[0]
				suite.Equal(v1.SearchCategory_IMAGE_INTEGRATIONS, searchResult.Category, "Category should be IMAGE_INTEGRATIONS")
				suite.Equal(integrations[2].GetId(), searchResult.Id, "Should return the correct integration ID")
				suite.Equal(integrations[2].GetName(), searchResult.Name, "Should return the correct integration name")

			case "SearchImageIntegrations - no matches":
				// Should return empty results - already verified by length check

			default:
				// For any other test cases, verify structure and data consistency
				for _, searchResult := range searchResults {
					suite.Equal(v1.SearchCategory_IMAGE_INTEGRATIONS, searchResult.Category, "Category should be IMAGE_INTEGRATIONS")
					suite.NotEmpty(searchResult.Id, "SearchResult should have an ID")
					suite.NotEmpty(searchResult.Name, "SearchResult should have a name")

					integration := integrationByID[searchResult.Id]
					suite.NotNil(integration, "Integration should exist")
					suite.Equal(integration.GetName(), searchResult.Name, "SearchResult name should match integration name")
				}
			}
		})
	}
}

func (suite *ImageIntegrationDataStoreTestSuite) TestSearchConsistencyBetweenMethods() {
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowAllAccessScopeChecker())

	// Generate test cluster ID as proper UUID
	consistencyClusterID := uuid.NewV4().String()

	// Create test integrations
	integrations := []*storage.ImageIntegration{
		{
			Id:        uuid.NewV4().String(),
			Name:      "consistency-test-1",
			ClusterId: consistencyClusterID,
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://consistency-1.com",
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "consistency-test-2",
			ClusterId: consistencyClusterID,
			IntegrationConfig: &storage.ImageIntegration_Quay{
				Quay: &storage.QuayConfig{
					Endpoint: "https://consistency-2.io",
				},
			},
		},
	}

	// Add all integrations
	for _, integration := range integrations {
		_, err := suite.datastore.AddImageIntegration(ctx, integration)
		suite.NoError(err)
	}

	// Test query that should match both integrations
	query := pkgSearch.NewQueryBuilder().AddStrings(pkgSearch.ClusterID, consistencyClusterID).ProtoQuery()

	// Get results from all three methods
	searchResults, err := suite.datastore.Search(ctx, query)
	suite.NoError(err)

	count, err := suite.datastore.Count(ctx, query)
	suite.NoError(err)

	searchImageIntegrations, err := suite.datastore.SearchImageIntegrations(ctx, query)
	suite.NoError(err)

	// Verify consistency between methods
	suite.Equal(len(searchResults), count, "Search result count should match Count result")
	suite.Equal(len(searchResults), len(searchImageIntegrations), "Search result count should match SearchImageIntegrations result count")
	suite.Equal(count, len(searchImageIntegrations), "Count should match SearchImageIntegrations result count")

	// Verify that the IDs returned by Search and SearchImageIntegrations are the same
	searchIDs := make(map[string]bool)
	for _, result := range searchResults {
		searchIDs[result.ID] = true
	}

	searchImageIntegrationIDs := make(map[string]bool)
	for _, result := range searchImageIntegrations {
		searchImageIntegrationIDs[result.Id] = true
	}

	suite.Equal(searchIDs, searchImageIntegrationIDs, "IDs returned by Search and SearchImageIntegrations should be identical")

	// Verify that the correct records are returned (should be both integrations from consistencyClusterID)
	suite.Len(searchResults, 2, "Should return exactly 2 results")
	for _, result := range searchResults {
		// Find the corresponding integration
		var foundIntegration *storage.ImageIntegration
		for _, integration := range integrations {
			if integration.GetId() == result.ID {
				foundIntegration = integration
				break
			}
		}
		suite.NotNil(foundIntegration, "Search result should correspond to a test integration")
		suite.Equal(consistencyClusterID, foundIntegration.GetClusterId(), "Integration should be from the consistency cluster")
	}

	// Verify SearchImageIntegrations returns correct data structure and content
	for _, searchResult := range searchImageIntegrations {
		suite.Equal(v1.SearchCategory_IMAGE_INTEGRATIONS, searchResult.Category, "Category should be IMAGE_INTEGRATIONS")

		// Find the corresponding integration
		var foundIntegration *storage.ImageIntegration
		for _, integration := range integrations {
			if integration.GetId() == searchResult.Id {
				foundIntegration = integration
				break
			}
		}
		suite.NotNil(foundIntegration, "SearchResult should correspond to a test integration")
		suite.Equal(foundIntegration.GetName(), searchResult.Name, "SearchResult name should match integration name")
		suite.Equal(consistencyClusterID, foundIntegration.GetClusterId(), "Integration should be from the consistency cluster")
	}
}

func (suite *ImageIntegrationDataStoreTestSuite) TestSearchWithAccessControlScenarios() {
	// Generate test cluster IDs as proper UUIDs
	sacCluster1ID := uuid.NewV4().String()
	sacCluster2ID := uuid.NewV4().String()

	// Test various access control scenarios with multiple integrations
	integrations := []*storage.ImageIntegration{
		{
			Id:        uuid.NewV4().String(),
			Name:      "sac-test-1",
			ClusterId: sacCluster1ID,
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://sac-test-1.com",
				},
			},
		},
		{
			Id:        uuid.NewV4().String(),
			Name:      "sac-test-2",
			ClusterId: sacCluster2ID,
			IntegrationConfig: &storage.ImageIntegration_Quay{
				Quay: &storage.QuayConfig{
					Endpoint: "https://sac-test-2.io",
				},
			},
		},
	}

	// Add integrations with write access
	for _, integration := range integrations {
		_, err := suite.datastore.AddImageIntegration(suite.hasWriteCtx, integration)
		suite.NoError(err)
	}

	testCases := []struct {
		name        string
		context     context.Context
		expectError bool
		description string
	}{
		{
			name:        "Search with no access",
			context:     suite.hasNoneCtx,
			expectError: false, // Search typically doesn't error but returns empty results
			description: "Search with no access should return empty results",
		},
		{
			name:        "Search with read access",
			context:     suite.hasReadCtx,
			expectError: false,
			description: "Search with read access should succeed",
		},
		{
			name:        "Search with write access",
			context:     suite.hasWriteCtx,
			expectError: false,
			description: "Search with write access should succeed",
		},
	}

	query := pkgSearch.EmptyQuery()

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Test Search method
			searchResults, err := suite.datastore.Search(tc.context, query)
			if tc.expectError {
				suite.Error(err, tc.description)
			} else {
				suite.NoError(err, tc.description)
				if tc.context == suite.hasNoneCtx {
					suite.Empty(searchResults, "Should return empty results with no access")
				} else {
					suite.NotEmpty(searchResults, "Should return results with read/write access")
					suite.Len(searchResults, 2, "Should return both test integrations")

					// Verify the correct records are returned
					returnedIDs := make(map[string]bool)
					for _, result := range searchResults {
						returnedIDs[result.ID] = true
					}
					// Should contain both integration IDs
					expectedIDs := []string{integrations[0].GetId(), integrations[1].GetId()}
					for _, expectedID := range expectedIDs {
						suite.True(returnedIDs[expectedID], "Should return integration %s", expectedID)
					}
				}
			}

			// Test Count method
			count, err := suite.datastore.Count(tc.context, query)
			if tc.expectError {
				suite.Error(err, tc.description)
			} else {
				suite.NoError(err, tc.description)
				if tc.context == suite.hasNoneCtx {
					suite.Zero(count, "Should return zero count with no access")
				} else {
					suite.Equal(2, count, "Should return count of 2 with read/write access")
				}
			}

			// Test SearchImageIntegrations method
			searchImageIntegrations, err := suite.datastore.SearchImageIntegrations(tc.context, query)
			if tc.expectError {
				suite.Error(err, tc.description)
			} else {
				suite.NoError(err, tc.description)
				if tc.context == suite.hasNoneCtx {
					suite.Empty(searchImageIntegrations, "Should return empty SearchResults with no access")
				} else {
					suite.NotEmpty(searchImageIntegrations, "Should return SearchResults with read/write access")
					suite.Len(searchImageIntegrations, 2, "Should return both test integrations")

					// Verify the correct records are returned
					integrationByID := make(map[string]*storage.ImageIntegration)
					for _, integration := range integrations {
						integrationByID[integration.GetId()] = integration
					}

					for _, searchResult := range searchImageIntegrations {
						suite.Equal(v1.SearchCategory_IMAGE_INTEGRATIONS, searchResult.Category, "Category should be IMAGE_INTEGRATIONS")

						integration := integrationByID[searchResult.Id]
						suite.NotNil(integration, "SearchResult should correspond to a test integration")
						suite.Equal(integration.GetName(), searchResult.Name, "SearchResult name should match integration name")
					}
				}
			}
		})
	}
}
