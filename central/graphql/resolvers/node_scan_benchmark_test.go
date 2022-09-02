package resolvers

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/require"
)

const (
	nodeOnlyQuery = `
 		query getNodes($query: String, $pagination: Pagination) {
 			nodes(query: $query, pagination: $pagination) { 
 				id
 			}}`

	nodeWithCountsQuery = `
 		query getNodes($query: String, $pagination: Pagination) {
 			nodes(query: $query, pagination: $pagination) { 
 				id
 				nodeComponentCount
 				nodeVulnerabilityCount
 			}}`

	nodeWithScanLongQuery = `
 		query getNodes($query: String, $pagination: Pagination) {
 			nodes(query: $query, pagination: $pagination) { 
 				id
 				scan {
 					nodeComponents {
 						name
 						lastScanned
 						nodeVulnerabilities {
 							cve
 							fixedByVersion
 						}
 					}
 				}
 			}}`

	nodeWithoutScanLongQuery = `
 		query getNodes($query: String, $pagination: Pagination) {
 			nodes(query: $query, pagination: $pagination) { 
 				id
 				nodeComponents {
 					name
 					lastScanned
 					nodeVulnerabilities {
 						cve
 						fixedByVersion
 					}
 				}
 			}}`
)

func BenchmarkNodeResolver(b *testing.B) {
	envIsolator := envisolator.NewEnvIsolator(b)
	envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")
	defer envIsolator.RestoreAll()

	if !features.PostgresDatastore.Enabled() {
		b.Skip("Skip postgres store tests")
		b.SkipNow()
	}

	mockCtrl := gomock.NewController(b)
	defer mockCtrl.Finish()
	db, gormDB := setupPostgresConn(b)
	defer pgtest.CloseGormDB(b, gormDB)
	defer db.Close()

	nodeDataStore := createNodeDatastoreForPostgres(b, mockCtrl, db, gormDB)
	nodeComponentDataStore := createNodeComponentDatastoreForPostgres(b, mockCtrl, db, gormDB)
	cveDataStore := createNodeCVEDatastoreForPostgres(b, db, gormDB)
	nodeComponentCVEEdgeDataStore := createNodeComponentCVEEdgeDatastoreForPostgres(b, db, gormDB)
	schema := setupResolverForNodeGraphQLTestsWithPostgres(b, nodeDataStore, nodeComponentDataStore, cveDataStore, nodeComponentCVEEdgeDataStore)
	ctx := contextWithNodePerm(b, mockCtrl)

	nodes := getTestNodesForPostgres(100)
	for _, node := range nodes {
		require.NoError(b, nodeDataStore.UpsertNode(ctx, node))
	}

	b.Run("GetNodeComponentsInNodeScanResolver", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				nodeWithScanQuery,
				"getNodes",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})

	b.Run("GetNodeComponentsWithoutNodeScanResolver", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				nodeWithoutScanQuery,
				"getNodes",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})

	b.Run("GetNodeComponentsDerivedFieldsWithNodeScanResolver", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				nodeWithScanLongQuery,
				"getNodes",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})

	b.Run("GetNodeComponentsDerivedWithoutNodeScanResolver", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				nodeWithoutScanLongQuery,
				"getNodes",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})

	b.Run("GetNodeOnly", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				nodeOnlyQuery,
				"getNodes",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})

	b.Run("GetNodeWithCounts", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				nodeWithCountsQuery,
				"getNodes",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})
}
