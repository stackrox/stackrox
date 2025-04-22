//go:build sql_integration

package reportgenerator

import (
	"context"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	namespaceDSMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	collectionSearch "github.com/stackrox/rox/central/resourcecollection/datastore/search"
	collectionPostgres "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	imagesView "github.com/stackrox/rox/central/views/images"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type FlatDataModelReportGeneratorBenchmarkTestSuite struct {
	b *testing.B

	ctx             context.Context
	testDB          *pgtest.TestPostgres
	reportGenerator *reportGeneratorImpl
	resolver        *resolvers.Resolver

	watchedImageDatastore watchedImageDS.DataStore

	clusterDatastore   *clusterDSMocks.MockDataStore
	namespaceDatastore *namespaceDSMocks.MockDataStore
}

func BenchmarkFlatDataModelReportGenerator(b *testing.B) {
	// TODO ROX-28898:enable feature flag by default to run the unit tests
	if !features.FlattenCVEData.Enabled() {
		b.Skip()
	}
	bts := &FlatDataModelReportGeneratorBenchmarkTestSuite{b: b}
	bts.setupTestSuite()

	clusters := []*storage.Cluster{
		{Id: uuid.NewV4().String(), Name: "c1"},
		{Id: uuid.NewV4().String(), Name: "c2"},
	}

	namespaces := testNamespaces(clusters, 10)

	deployments, images := testDeploymentsWithImages(namespaces, 100)
	bts.upsertManyImages(images)
	bts.upsertManyDeployments(deployments)

	watchedImages := testWatchedImages(500)
	bts.upsertManyImages(watchedImages)
	bts.upsertManyWatchedImages(watchedImages)

	bts.clusterDatastore.EXPECT().GetClusters(gomock.Any()).
		Return(clusters, nil).AnyTimes()

	bts.namespaceDatastore.EXPECT().GetAllNamespaces(gomock.Any()).
		Return(namespaces, nil).AnyTimes()

	collection := testCollection("col4", "", "", "")
	fixability := storage.VulnerabilityReportFilters_BOTH
	severities := allSeverities()
	imageTypes := []storage.VulnerabilityReportFilters_ImageType{
		storage.VulnerabilityReportFilters_DEPLOYED,
		storage.VulnerabilityReportFilters_WATCHED,
	}

	expectedRowCount := 5000

	reportSnap := testReportSnapshot(collection.GetId(), fixability, severities, imageTypes, nil)

	b.Run("GetReportDataSQF", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reportData, err := bts.reportGenerator.getReportDataSQF(reportSnap, collection, time.Time{})
			require.NoError(b, err)
			require.Equal(b, expectedRowCount, len(reportData.CVEResponses))
		}
	})
}

func (bts *FlatDataModelReportGeneratorBenchmarkTestSuite) setupTestSuite() {

	bts.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(bts.b)
	bts.testDB = pgtest.ForT(bts.b)

	// set up tables
	imageDataStore := resolvers.CreateTestImageV2Datastore(bts.b, bts.testDB, mockCtrl)
	bts.watchedImageDatastore = watchedImageDS.GetTestPostgresDataStore(bts.b, bts.testDB.DB)
	var schema *graphql.Schema
	bts.resolver, schema = resolvers.SetupTestResolver(bts.b,
		imagesView.NewImageView(bts.testDB.DB),
		imageDataStore,
		resolvers.CreateTestImageComponentV2Datastore(bts.b, bts.testDB, mockCtrl),
		resolvers.CreateTestImageCVEV2Datastore(bts.b, bts.testDB),
		resolvers.CreateTestDeploymentDatastore(bts.b, bts.testDB, mockCtrl, imageDataStore),
	)
	collectionStore := collectionPostgres.CreateTableAndNewStore(bts.ctx, bts.testDB.DB, bts.testDB.GetGormDB(bts.b))
	_, collectionQueryResolver, err := collectionDS.New(collectionStore, collectionSearch.New(collectionStore))
	require.NoError(bts.b, err)
	bts.clusterDatastore = clusterDSMocks.NewMockDataStore(mockCtrl)
	bts.namespaceDatastore = namespaceDSMocks.NewMockDataStore(mockCtrl)

	bts.reportGenerator = newReportGeneratorImpl(bts.testDB, nil, bts.resolver.DeploymentDataStore,
		bts.watchedImageDatastore, collectionQueryResolver, nil, nil, bts.clusterDatastore,
		bts.namespaceDatastore, bts.resolver.ImageCVEDataStore, bts.resolver.ImageCVEV2DataStore, schema)
}

func (bts *FlatDataModelReportGeneratorBenchmarkTestSuite) upsertManyImages(images []*storage.Image) {
	for _, img := range images {
		err := bts.resolver.ImageDataStore.UpsertImage(bts.ctx, img)
		require.NoError(bts.b, err)
	}
}

func (bts *FlatDataModelReportGeneratorBenchmarkTestSuite) upsertManyWatchedImages(images []*storage.Image) {
	for _, img := range images {
		err := bts.watchedImageDatastore.UpsertWatchedImage(bts.ctx, img.Name.FullName)
		require.NoError(bts.b, err)
	}
}

func (bts *FlatDataModelReportGeneratorBenchmarkTestSuite) upsertManyDeployments(deployments []*storage.Deployment) {
	for _, dep := range deployments {
		err := bts.resolver.DeploymentDataStore.UpsertDeployment(bts.ctx, dep)
		require.NoError(bts.b, err)
	}
}
