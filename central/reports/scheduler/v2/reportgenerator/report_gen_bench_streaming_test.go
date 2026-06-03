//go:build sql_integration

package reportgenerator

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	namespaceDSMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	collectionPostgres "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	imagesView "github.com/stackrox/rox/central/views/images"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var reuseDB = true

func BenchmarkFullReportPipeline(b *testing.B) {
	ctx := loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(b)

	var testDB *pgtest.TestPostgres
	if reuseDB {
		testDB = benchGetOrCreateDB(b, "bench_report_pipeline")
	} else {
		testDB = pgtest.ForT(b)
	}

	watchedImageDatastore := watchedImageDS.GetTestPostgresDataStore(b, testDB.DB)
	var schema *graphql.Schema
	var resolver *resolvers.Resolver
	if features.FlattenImageData.Enabled() {
		imgV2DataStore := resolvers.CreateTestImageV2Datastore(b, testDB, mockCtrl)
		resolver, schema = resolvers.SetupTestResolver(b,
			imagesView.NewImageView(testDB.DB),
			imgV2DataStore,
			resolvers.CreateTestImageComponentV2Datastore(b, testDB, mockCtrl),
			resolvers.CreateTestImageCVEV2Datastore(b, testDB),
			resolvers.CreateTestDeploymentDatastoreWithImageV2(b, testDB, mockCtrl, imgV2DataStore),
			deploymentsView.NewDeploymentView(testDB.DB),
		)
	} else {
		imageDataStore := resolvers.CreateTestImageDatastore(b, testDB, mockCtrl)
		resolver, schema = resolvers.SetupTestResolver(b,
			imagesView.NewImageView(testDB.DB),
			imageDataStore,
			resolvers.CreateTestImageComponentV2Datastore(b, testDB, mockCtrl),
			resolvers.CreateTestImageCVEV2Datastore(b, testDB),
			resolvers.CreateTestDeploymentDatastore(b, testDB, mockCtrl, imageDataStore),
			deploymentsView.NewDeploymentView(testDB.DB),
		)
	}

	collectionStore := collectionPostgres.New(testDB)
	_, collectionQueryResolver, err := collectionDS.New(collectionStore)
	require.NoError(b, err)
	clusterDatastore := clusterDSMocks.NewMockDataStore(mockCtrl)
	namespaceDatastore := namespaceDSMocks.NewMockDataStore(mockCtrl)

	reportGenerator := newReportGeneratorImpl(testDB, nil, resolver.DeploymentDataStore,
		watchedImageDatastore, collectionQueryResolver, nil, nil, clusterDatastore,
		namespaceDatastore, resolver.ImageCVEV2DataStore, schema)

	clusters := []*storage.Cluster{
		{Id: uuid.NewV4().String(), Name: "c1"},
		{Id: uuid.NewV4().String(), Name: "c2"},
		{Id: uuid.NewV4().String(), Name: "c3"},
		{Id: uuid.NewV4().String(), Name: "c4"},
		{Id: uuid.NewV4().String(), Name: "c5"},
	}

	namespaces := testNamespaces(clusters, 100)

	// Check if test data already exists (for DB reuse).
	depCount, err := resolver.DeploymentDataStore.Count(ctx, search.EmptyQuery())
	require.NoError(b, err)
	dataExists := depCount > 0

	if !dataExists {
		for _, ns := range namespaces {
			nsDS, nsErr := namespaceDS.GetTestPostgresDataStore(b, testDB.DB)
			require.NoError(b, nsErr)
			require.NoError(b, nsDS.AddNamespace(ctx, ns))
		}

		deployments, images := testDeploymentsWithImages(namespaces, 100)
		if features.FlattenImageData.Enabled() {
			for _, img := range images {
				require.NoError(b, resolver.ImageV2DataStore.UpsertImage(ctx, imageUtils.ConvertToV2(img)))
			}
		} else {
			for _, img := range images {
				require.NoError(b, resolver.ImageDataStore.UpsertImage(ctx, img))
			}
		}
		for _, dep := range deployments {
			require.NoError(b, resolver.DeploymentDataStore.UpsertDeployment(ctx, dep))
		}

		watchedImages := testWatchedImages(500)
		if features.FlattenImageData.Enabled() {
			for _, img := range watchedImages {
				require.NoError(b, resolver.ImageV2DataStore.UpsertImage(ctx, imageUtils.ConvertToV2(img)))
			}
		} else {
			for _, img := range watchedImages {
				require.NoError(b, resolver.ImageDataStore.UpsertImage(ctx, img))
			}
		}
		for _, img := range watchedImages {
			require.NoError(b, watchedImageDatastore.UpsertWatchedImage(ctx, img.GetName().GetFullName()))
		}
		b.Log("Inserted test data")
	} else {
		b.Log("Reusing existing test data")
	}

	clusterDatastore.EXPECT().GetClusters(gomock.Any()).
		Return(clusters, nil).AnyTimes()
	namespaceDatastore.EXPECT().GetAllNamespaces(gomock.Any()).
		Return(namespaces, nil).AnyTimes()

	collection := testCollection("bench_col", "", "", "")
	imageTypes := []storage.VulnerabilityReportFilters_ImageType{
		storage.VulnerabilityReportFilters_DEPLOYED,
		storage.VulnerabilityReportFilters_WATCHED,
	}
	reportSnap := testReportSnapshot(collection.GetId(),
		storage.VulnerabilityReportFilters_BOTH, allSeverities(), imageTypes, nil)

	// 5 clusters × 100 ns × 100 deps × 2 CVEs + 500 watched × 2 CVEs = 101000
	expectedRowCount := 101000

	b.Run("Old_Buffered", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var peakHeap atomic.Uint64
			done := make(chan struct{})
			go trackPeakHeap(&peakHeap, done)

			reportData, err := reportGenerator.getReportDataSQF(reportSnap, collection, time.Time{})
			require.NoError(b, err)
			require.Equal(b, expectedRowCount, len(reportData.CVEResponses))

			buf, err := GenerateCSV(reportData.CVEResponses, "bench_test")
			require.NoError(b, err)
			require.True(b, buf.Len() > 0)

			close(done)
			b.ReportMetric(float64(peakHeap.Load()), "peak_heap_bytes")
		}
	})

	b.Run("New_Streaming", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var peakHeap atomic.Uint64
			done := make(chan struct{})
			go trackPeakHeap(&peakHeap, done)

			result, err := reportGenerator.generateReportStreamingSQF(reportSnap, collection, time.Time{}, "bench_test")
			require.NoError(b, err)
			require.Equal(b, expectedRowCount, result.NumDeployedImageResults+result.NumWatchedImageResults)
			require.True(b, result.ZippedCSVData.Len() > 0)

			close(done)
			b.ReportMetric(float64(peakHeap.Load()), "peak_heap_bytes")
		}
	})
}

// benchGetOrCreateDB returns a TestPostgres using a fixed DB name. On the
// first run it creates the DB and applies schemas; on subsequent runs (when
// ROX_POSTGRES_TEST_KEEP_DB=true) it reuses the existing DB and skips schema
// creation, cutting setup from minutes to seconds.
func benchGetOrCreateDB(b *testing.B, dbName string) *pgtest.TestPostgres {
	pgtest.CreateDatabase(b, dbName)
	source := pgtest.GetConnectionStringWithDatabaseName(b, dbName)
	gormDB := pgtest.OpenGormDB(b, source)
	pkgSchema.ApplyAllSchemasIncludingTests(context.Background(), gormDB, b)
	pgtest.CloseGormDB(b, gormDB)
	pool := pgtest.ForTCustomPool(b, dbName)
	return &pgtest.TestPostgres{
		DB: pool,
	}
}

func trackPeakHeap(peak *atomic.Uint64, done <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			for {
				cur := peak.Load()
				if m.HeapAlloc <= cur || peak.CompareAndSwap(cur, m.HeapAlloc) {
					break
				}
			}
		}
	}
}
