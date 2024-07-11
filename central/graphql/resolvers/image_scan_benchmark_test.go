package resolvers

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	imageOnlyQuery = `
		query getImages($query: String, $pagination: Pagination) {
			images(query: $query, pagination: $pagination) { 
				id
			}}`

	imageWithCountsQuery = `
		query getImages($query: String, $pagination: Pagination) {
			images(query: $query, pagination: $pagination) { 
				id
				imageComponentCount
				imageVulnerabilityCount
			}}`

	imageWithScanLongQuery = `
		query getImages($query: String, $pagination: Pagination) {
			images(query: $query, pagination: $pagination) { 
				id
				scan {
					imageComponents {
						name
						lastScanned
						imageVulnerabilities {
							cve
							fixedByVersion
						}
					}
				}
			}}`

	imageWithoutScanLongQuery = `
		query getImages($query: String, $pagination: Pagination) {
			images(query: $query, pagination: $pagination) { 
				id
				imageComponents {
					name
					lastScanned
					imageVulnerabilities {
						cve
						fixedByVersion
					}
				}
			}}`

	imageWithTopLevelScanTimeQuery = `
		query getImages($query: String, $pagination: Pagination) {
			images(query: $query, pagination: $pagination) { 
				id
				scanTime
			}}`

	imageWithNestedScanTimeQuery = `
		query getImages($query: String, $pagination: Pagination) {
			images(query: $query, pagination: $pagination) { 
				id
				scan {
					scanTime
				}
			}}`
)

func BenchmarkImageResolver(b *testing.B) {

	mockCtrl := gomock.NewController(b)
	testDB := SetupTestPostgresConn(b)
	defer testDB.Teardown(b)

	resolver, schema := SetupTestResolver(b,
		CreateTestImageDatastore(b, testDB, mockCtrl),
		CreateTestImageComponentDatastore(b, testDB, mockCtrl),
		CreateTestImageCVEDatastore(b, testDB),
		CreateTestImageComponentCVEEdgeDatastore(b, testDB),
	)
	ctx := contextWithImagePerm(b, mockCtrl)

	images := getTestImages(100)
	for _, image := range images {
		require.NoError(b, resolver.ImageDataStore.UpsertImage(ctx, image))
	}

	b.Run("GetImageComponentsInImageScanResolver", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				imageWithScanQuery,
				"getImages",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})

	b.Run("GetImageComponentsWithoutImageScanResolver", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				imageWithoutScanQuery,
				"getImages",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})

	b.Run("GetImageComponentsDerivedFieldsWithImageScanResolver", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				imageWithScanLongQuery,
				"getImages",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})

	b.Run("GetImageComponentsDerivedWithoutImageScanResolver", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				imageWithoutScanLongQuery,
				"getImages",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})

	b.Run("GetImageOnly", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				imageOnlyQuery,
				"getImages",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})

	b.Run("GetImageWithCounts", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				imageWithCountsQuery,
				"getImages",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})

	b.Run("GetImageScanTimeTopLevel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				imageWithTopLevelScanTimeQuery,
				"getImages",
				map[string]interface{}{
					"pagination": map[string]interface{}{
						"limit": 25,
					},
				},
			)
			require.Len(b, response.Errors, 0)
		}
	})

	b.Run("GetImageScanTimeNested", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			response := schema.Exec(ctx,
				imageWithNestedScanTimeQuery,
				"getImages",
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
