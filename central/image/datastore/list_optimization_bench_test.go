//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/image/datastore/keyfence"
	pgStoreV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"go.uber.org/mock/gomock"
)

// BenchmarkSearchListImages compares performance of legacy vs optimized paths
func BenchmarkSearchListImages(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)

	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	dbStore := pgStoreV2.New(testDB.DB, false, keyfence.ImageKeyFenceSingleton())
	datastore := NewWithPostgres(dbStore, mockRisk, ranking.ImageRanker(), ranking.ComponentRanker())

	// Setup: Insert varying numbers of test images
	imageCounts := []int{10, 100, 1000}

	for _, count := range imageCounts {
		b.Run(fmt.Sprintf("Images_%d", count), func(b *testing.B) {
			// Clean database
			_, err := testDB.DB.Exec(ctx, "TRUNCATE images_v2 CASCADE")
			if err != nil {
				b.Fatal(err)
			}

			// Insert test images with realistic scan data
			for i := 0; i < count; i++ {
				img := fixtures.GetImage()
				img.Id = fmt.Sprintf("sha-%d", i)
				img.Name.FullName = fmt.Sprintf("test/image-%d:v1", i)
				img.SetComponents = &storage.Image_Components{Components: int32(10 + i%100)}
				img.SetCves = &storage.Image_Cves{Cves: int32(5 + i%50)}
				img.SetFixable = &storage.Image_FixableCves{FixableCves: int32(i % 25)}

				if err := datastore.UpsertImage(ctx, img); err != nil {
					b.Fatal(err)
				}
			}

			query := pkgSearch.EmptyQuery()

			// Benchmark legacy path
			b.Run("Legacy", func(b *testing.B) {
				b.Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "false")
				// Feature flag controlled via b.Setenv above

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					results, err := datastore.SearchListImages(ctx, query)
					if err != nil {
						b.Fatal(err)
					}
					if len(results) != count {
						b.Fatalf("Expected %d results, got %d", count, len(results))
					}
				}
			})

			// Benchmark optimized path
			b.Run("Optimized", func(b *testing.B) {
				b.Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "true")
				// Feature flag controlled via b.Setenv above

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					results, err := datastore.SearchListImages(ctx, query)
					if err != nil {
						b.Fatal(err)
					}
					if len(results) != count {
						b.Fatalf("Expected %d results, got %d", count, len(results))
					}
				}
			})
		})
	}
}

// BenchmarkSearchListImagesWithPagination tests pagination performance
func BenchmarkSearchListImagesWithPagination(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)

	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	dbStore := pgStoreV2.New(testDB.DB, false, keyfence.ImageKeyFenceSingleton())
	datastore := NewWithPostgres(dbStore, mockRisk, ranking.ImageRanker(), ranking.ComponentRanker())

	// Insert 1000 test images
	imageCount := 1000
	for i := 0; i < imageCount; i++ {
		img := fixtures.GetImage()
		img.Id = fmt.Sprintf("sha-page-%d", i)
		img.Name.FullName = fmt.Sprintf("test/page-%d:v1", i)
		img.SetComponents = &storage.Image_Components{Components: int32(10 + i%100)}
		img.SetCves = &storage.Image_Cves{Cves: int32(5 + i%50)}
		img.SetFixable = &storage.Image_FixableCves{FixableCves: int32(i % 25)}

		if err := datastore.UpsertImage(ctx, img); err != nil {
			b.Fatal(err)
		}
	}

	pageSizes := []int{10, 50, 100}

	for _, pageSize := range pageSizes {
		b.Run(fmt.Sprintf("PageSize_%d", pageSize), func(b *testing.B) {
			query := pkgSearch.NewQueryBuilder().
				WithPagination(pkgSearch.NewPagination().Limit(int32(pageSize))).
				ProtoQuery()

			// Benchmark legacy path
			b.Run("Legacy", func(b *testing.B) {
				b.Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "false")
				// Feature flag controlled via b.Setenv above

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					results, err := datastore.SearchListImages(ctx, query)
					if err != nil {
						b.Fatal(err)
					}
					if len(results) > pageSize {
						b.Fatalf("Expected max %d results, got %d", pageSize, len(results))
					}
				}
			})

			// Benchmark optimized path
			b.Run("Optimized", func(b *testing.B) {
				b.Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "true")
				// Feature flag controlled via b.Setenv above

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					results, err := datastore.SearchListImages(ctx, query)
					if err != nil {
						b.Fatal(err)
					}
					if len(results) > pageSize {
						b.Fatalf("Expected max %d results, got %d", pageSize, len(results))
					}
				}
			})
		})
	}
}

// BenchmarkSearchListImagesWithFilter tests filtered query performance
func BenchmarkSearchListImagesWithFilter(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)

	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	dbStore := pgStoreV2.New(testDB.DB, false, keyfence.ImageKeyFenceSingleton())
	datastore := NewWithPostgres(dbStore, mockRisk, ranking.ImageRanker(), ranking.ComponentRanker())

	// Insert 1000 test images with varying prefixes
	imageCount := 1000
	for i := 0; i < imageCount; i++ {
		img := fixtures.GetImage()
		img.Id = fmt.Sprintf("sha-filter-%d", i)
		prefix := fmt.Sprintf("prefix%d", i%10) // 10 different prefixes
		img.Name.FullName = fmt.Sprintf("%s/image-%d:v1", prefix, i)
		img.SetComponents = &storage.Image_Components{Components: int32(10 + i%100)}
		img.SetCves = &storage.Image_Cves{Cves: int32(5 + i%50)}
		img.SetFixable = &storage.Image_FixableCves{FixableCves: int32(i % 25)}

		if err := datastore.UpsertImage(ctx, img); err != nil {
			b.Fatal(err)
		}
	}

	// Filter for images with prefix0 (should match ~100 images)
	query := pkgSearch.NewQueryBuilder().
		AddStrings(pkgSearch.ImageName, "r/prefix0").
		ProtoQuery()

	// Benchmark legacy path
	b.Run("Legacy", func(b *testing.B) {
		b.Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "false")
		// Feature flag controlled via b.Setenv above

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchListImages(ctx, query)
			if err != nil {
				b.Fatal(err)
			}
			_ = results
		}
	})

	// Benchmark optimized path
	b.Run("Optimized", func(b *testing.B) {
		b.Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "true")
		// Feature flag controlled via b.Setenv above

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchListImages(ctx, query)
			if err != nil {
				b.Fatal(err)
			}
			_ = results
		}
	})
}

// BenchmarkMemoryAllocation measures memory allocations
func BenchmarkMemoryAllocation(b *testing.B) {
	ctx := sac.WithAllAccess(context.Background())
	testDB := pgtest.ForT(b)

	mockRisk := mockRisks.NewMockDataStore(gomock.NewController(b))
	dbStore := pgStoreV2.New(testDB.DB, false, keyfence.ImageKeyFenceSingleton())
	datastore := NewWithPostgres(dbStore, mockRisk, ranking.ImageRanker(), ranking.ComponentRanker())

	// Insert 100 test images with large scan data
	imageCount := 100
	for i := 0; i < imageCount; i++ {
		img := fixtures.GetImage()
		img.Id = fmt.Sprintf("sha-mem-%d", i)
		img.Name.FullName = fmt.Sprintf("test/memory-%d:v1", i)
		img.SetComponents = &storage.Image_Components{Components: int32(100)}
		img.SetCves = &storage.Image_Cves{Cves: int32(50)}
		img.SetFixable = &storage.Image_FixableCves{FixableCves: int32(25)}

		if err := datastore.UpsertImage(ctx, img); err != nil {
			b.Fatal(err)
		}
	}

	query := pkgSearch.EmptyQuery()

	// Benchmark legacy path memory
	b.Run("Legacy", func(b *testing.B) {
		b.Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "false")
		// Feature flag controlled via b.Setenv above

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchListImages(ctx, query)
			if err != nil {
				b.Fatal(err)
			}
			_ = results
		}
	})

	// Benchmark optimized path memory
	b.Run("Optimized", func(b *testing.B) {
		b.Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "true")
		// Feature flag controlled via b.Setenv above

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results, err := datastore.SearchListImages(ctx, query)
			if err != nil {
				b.Fatal(err)
			}
			_ = results
		}
	})
}
