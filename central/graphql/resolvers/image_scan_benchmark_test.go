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
)

func BenchmarkImageResolver(b *testing.B) {
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

	imageDataStore := createImageDatastore(b, mockCtrl, db, gormDB)
	imageComponentDataStore := createImageComponentDatastore(b, mockCtrl, db, gormDB)
	cveDataStore := createImageCVEDatastore(b, db, gormDB)
	componentCVEEdgeDataStore := createImageComponentCVEEdgeDatastore(b, db, gormDB)
	resolver, schema := setupResolver(b, imageDataStore, imageComponentDataStore, cveDataStore, componentCVEEdgeDataStore)
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
}
