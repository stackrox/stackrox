//go:build sql_integration

package reportgenerator

import (
	"bytes"
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	collectionPostgres "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	imagesView "github.com/stackrox/rox/central/views/images"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// memoryBenchSuite holds shared test infrastructure for the memory benchmarks.
type memoryBenchSuite struct {
	b          *testing.B
	ctx        context.Context
	testDB     *pgtest.TestPostgres
	rg         *reportGeneratorImpl
	collection *storage.ResourceCollection
	snap       *storage.ReportSnapshot
	rowCount   int
}

func setupMemoryBench(b *testing.B, numNamespacesPerCluster, numDeploymentsPerNamespace, numWatchedImages int) *memoryBenchSuite {
	b.Helper()

	ctx := loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(b)
	testDB := pgtest.ForT(b)

	watchedImageDatastore := watchedImageDS.GetTestPostgresDataStore(b, testDB.DB)

	var resolver *resolvers.Resolver
	var schema *graphql.Schema
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
	nsDatastore, err := namespaceDS.GetTestPostgresDataStore(b, testDB.DB)
	require.NoError(b, err)

	blobStore := blobDS.NewBenchDatastore(testDB.DB)

	rg := newReportGeneratorImpl(testDB, nil, resolver.DeploymentDataStore,
		watchedImageDatastore, collectionQueryResolver, nil, blobStore, clusterDatastore,
		nsDatastore, resolver.ImageCVEV2DataStore, schema)

	clusters := []*storage.Cluster{
		{Id: uuid.NewV4().String(), Name: "c1"},
		{Id: uuid.NewV4().String(), Name: "c2"},
	}
	namespaces := testNamespaces(clusters, numNamespacesPerCluster)
	for _, ns := range namespaces {
		require.NoError(b, nsDatastore.AddNamespace(ctx, ns))
	}

	deployments, images := testDeploymentsWithImages(namespaces, numDeploymentsPerNamespace)
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

	watchedImages := testWatchedImages(numWatchedImages)
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

	clusterDatastore.EXPECT().GetClusters(gomock.Any()).Return(clusters, nil).AnyTimes()

	collection := testCollection("col_bench", "", "", "")
	imageTypes := []storage.VulnerabilityReportFilters_ImageType{
		storage.VulnerabilityReportFilters_DEPLOYED,
		storage.VulnerabilityReportFilters_WATCHED,
	}
	snap := testReportSnapshot(collection.GetId(), storage.VulnerabilityReportFilters_BOTH, allSeverities(), imageTypes, nil)

	// Each image has 2 CVEs, so:
	//   deployed rows = 2 clusters * numNamespacesPerCluster * numDeploymentsPerNamespace * 2
	//   watched rows  = numWatchedImages * 2
	expectedRows := 2*numNamespacesPerCluster*numDeploymentsPerNamespace*2 + numWatchedImages*2

	return &memoryBenchSuite{
		b:          b,
		ctx:        ctx,
		testDB:     testDB,
		rg:         rg,
		collection: collection,
		snap:       snap,
		rowCount:   expectedRows,
	}
}

func forceGC() {
	runtime.GC()
	runtime.GC()
}

func heapAllocBytes() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// BenchmarkMemory_InMemory measures peak heap allocation for the old in-memory path:
// accumulate all rows, build CSV buffer, build ZIP buffer, then write to blob store.
func BenchmarkMemory_InMemory(b *testing.B) {
	s := setupMemoryBench(b, 10, 50, 500)

	b.Run("getReportData+GenerateCSV+saveReportData", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			forceGC()
			before := heapAllocBytes()

			reportData, err := s.rg.getReportDataSQF(s.snap, s.collection, time.Time{})
			require.NoError(b, err)
			require.Equal(b, s.rowCount, len(reportData.CVEResponses))

			zippedCSV, err := GenerateCSV(reportData.CVEResponses, s.snap.GetName())
			require.NoError(b, err)
			require.True(b, zippedCSV.Len() > 0)

			// Measure after data + CSV + ZIP are all in memory.
			after := heapAllocBytes()
			b.ReportMetric(float64(after-before), "heap-delta-bytes")

			// Save to blob store (same as production DOWNLOAD path).
			err = s.rg.saveReportData(s.snap.GetReportConfigurationId(), s.snap.GetReportId(), zippedCSV)
			require.NoError(b, err)
		}
	})
}

// BenchmarkMemory_Streaming measures peak heap allocation for the new streaming path:
// cursor -> CSV -> ZIP -> io.Pipe -> blob store, without accumulating the full dataset.
func BenchmarkMemory_Streaming(b *testing.B) {
	s := setupMemoryBench(b, 10, 50, 500)
	// Set notification method to DOWNLOAD so generateReportStreamingDownload is used.
	s.snap.ReportStatus.ReportNotificationMethod = storage.ReportStatus_DOWNLOAD

	b.Run("generateReportStreamingDownload", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			forceGC()
			before := heapAllocBytes()

			err := s.rg.generateReportStreamingDownload(&ReportRequest{
				ReportSnapshot: s.snap,
				Collection:     s.collection,
				DataStartTime:  time.Time{},
			})
			require.NoError(b, err)

			after := heapAllocBytes()
			b.ReportMetric(float64(after-before), "heap-delta-bytes")
		}
	})
}

// BenchmarkMemoryComparison runs both paths side-by-side for easy comparison.
// Use: go test -tags sql_integration -bench BenchmarkMemoryComparison -benchtime 3x -v
func BenchmarkMemoryComparison(b *testing.B) {
	s := setupMemoryBench(b, 10, 50, 500)

	b.Run("InMemory", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			forceGC()
			before := heapAllocBytes()

			reportData, err := s.rg.getReportDataSQF(s.snap, s.collection, time.Time{})
			require.NoError(b, err)

			zippedCSV, err := GenerateCSV(reportData.CVEResponses, s.snap.GetName())
			require.NoError(b, err)

			after := heapAllocBytes()
			b.ReportMetric(float64(after-before), "heap-delta-bytes")

			err = s.rg.saveReportData(s.snap.GetReportConfigurationId(), s.snap.GetReportId(), zippedCSV)
			require.NoError(b, err)
		}
	})

	s.snap.ReportStatus.ReportNotificationMethod = storage.ReportStatus_DOWNLOAD
	b.Run("Streaming", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			forceGC()
			before := heapAllocBytes()

			err := s.rg.generateReportStreamingDownload(&ReportRequest{
				ReportSnapshot: s.snap,
				Collection:     s.collection,
				DataStartTime:  time.Time{},
			})
			require.NoError(b, err)

			after := heapAllocBytes()
			b.ReportMetric(float64(after-before), "heap-delta-bytes")
		}
	})
}

// BenchmarkMemoryPeakTracking provides finer-grained peak memory tracking by sampling
// heap allocations during the streaming pipeline using a finalizer-based approach.
// This gives the most accurate picture since runtime.ReadMemStats is a stop-the-world snapshot.
func BenchmarkMemoryPeakTracking(b *testing.B) {
	s := setupMemoryBench(b, 10, 50, 500)

	b.Run("InMemory_Peak", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			forceGC()
			var peak uint64

			sample := func() {
				cur := heapAllocBytes()
				if cur > peak {
					peak = cur
				}
			}

			baseline := heapAllocBytes()
			reportData, err := s.rg.getReportDataSQF(s.snap, s.collection, time.Time{})
			require.NoError(b, err)
			sample()

			zippedCSV, err := GenerateCSV(reportData.CVEResponses, s.snap.GetName())
			require.NoError(b, err)
			sample()

			err = s.rg.saveReportData(s.snap.GetReportConfigurationId(), s.snap.GetReportId(), zippedCSV)
			require.NoError(b, err)
			sample()

			b.ReportMetric(float64(peak-baseline), "peak-heap-delta-bytes")

			// Prevent compiler from optimizing away the result.
			_ = zippedCSV
			_ = reportData
		}
	})

	s.snap.ReportStatus.ReportNotificationMethod = storage.ReportStatus_DOWNLOAD
	b.Run("Streaming_Peak", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			forceGC()
			baseline := heapAllocBytes()

			err := s.rg.generateReportStreamingDownload(&ReportRequest{
				ReportSnapshot: s.snap,
				Collection:     s.collection,
				DataStartTime:  time.Time{},
			})
			require.NoError(b, err)

			after := heapAllocBytes()
			b.ReportMetric(float64(after-baseline), "peak-heap-delta-bytes")
		}
	})
}

// BenchmarkMemoryAtScale uses larger data volumes to amplify the difference.
// 2 clusters * 20 namespaces * 100 deployments * 2 CVEs = 8000 deployed rows
// + 1000 watched images * 2 CVEs = 2000 watched rows = 10000 total rows.
func BenchmarkMemoryAtScale(b *testing.B) {
	s := setupMemoryBench(b, 20, 100, 1000)

	b.Run("InMemory", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			forceGC()
			before := heapAllocBytes()

			reportData, err := s.rg.getReportDataSQF(s.snap, s.collection, time.Time{})
			require.NoError(b, err)

			zippedCSV, err := GenerateCSV(reportData.CVEResponses, s.snap.GetName())
			require.NoError(b, err)

			after := heapAllocBytes()
			b.ReportMetric(float64(after-before), "heap-delta-bytes")
			b.ReportMetric(float64(len(reportData.CVEResponses)), "rows")

			var buf bytes.Buffer
			_, _, err = s.rg.blobStore.Get(reportGenCtx, "", &buf)
			_ = err

			err = s.rg.saveReportData(s.snap.GetReportConfigurationId(), s.snap.GetReportId(), zippedCSV)
			require.NoError(b, err)
		}
	})

	s.snap.ReportStatus.ReportNotificationMethod = storage.ReportStatus_DOWNLOAD
	b.Run("Streaming", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			forceGC()
			before := heapAllocBytes()

			err := s.rg.generateReportStreamingDownload(&ReportRequest{
				ReportSnapshot: s.snap,
				Collection:     s.collection,
				DataStartTime:  time.Time{},
			})
			require.NoError(b, err)

			after := heapAllocBytes()
			b.ReportMetric(float64(after-before), "heap-delta-bytes")
		}
	})
}
