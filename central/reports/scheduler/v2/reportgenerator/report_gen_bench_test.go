//go:build sql_integration

package reportgenerator

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	namespaceDSMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	collectionSearch "github.com/stackrox/rox/central/resourcecollection/datastore/search"
	collectionPostgres "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type ReportGeneratorBenchmarkTestSuite struct {
	b        *testing.B
	mockCtrl *gomock.Controller

	ctx             context.Context
	testDB          *pgtest.TestPostgres
	reportGenerator *reportGeneratorImpl
	resolver        *resolvers.Resolver
	schema          *graphql.Schema

	watchedImageDatastore   watchedImageDS.DataStore
	collectionQueryResolver collectionDS.QueryResolver

	clusterDatastore   *clusterDSMocks.MockDataStore
	namespaceDatastore *namespaceDSMocks.MockDataStore
}

func BenchmarkReportGenerator(b *testing.B) {
	s.T().Setenv(features.VulnReportingEnhancements.EnvVar(), "true")
	if !features.VulnReportingEnhancements.Enabled() {
		s.T().Skip("Skip tests when ROX_VULN_MGMT_REPORTING_ENHANCEMENTS disabled")
		s.T().SkipNow()
	}

	bts := &ReportGeneratorBenchmarkTestSuite{b: b}
	bts.setupTestSuite()
	defer bts.teardownTestSuite()

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
	expectedDeploymentCount := 2000
	expectedWatchedImageCount := 500

	reportSnap := testReportSnapshot(collection.GetId(), fixability, severities, imageTypes, nil)

	b.Run("GetReportDataGraphQL", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			deployedImgResults, watchedImgResults, err := bts.reportGenerator.getReportData(reportSnap, collection, nil)
			require.NoError(b, err)
			require.Equal(b, expectedDeploymentCount, len(deployedImgResults[0].Deployments))
			require.Equal(b, expectedWatchedImageCount, len(watchedImgResults[0].Images))
		}
	})

	b.Run("GetReportDataSQF", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reportData, err := bts.reportGenerator.getReportDataSQF(reportSnap, collection, nil)
			require.NoError(b, err)
			require.Equal(b, expectedRowCount, len(reportData.CVEResponses))
		}
	})
}

func (bts *ReportGeneratorBenchmarkTestSuite) setupTestSuite() {
	bts.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	bts.mockCtrl = gomock.NewController(bts.b)
	bts.testDB = resolvers.SetupTestPostgresConn(bts.b)

	imageDataStore := resolvers.CreateTestImageDatastore(bts.b, bts.testDB, bts.mockCtrl)
	imageCVEDatastore := resolvers.CreateTestImageCVEDatastore(bts.b, bts.testDB)
	bts.resolver, bts.schema = resolvers.SetupTestResolver(bts.b,
		imageDataStore,
		resolvers.CreateTestImageComponentDatastore(bts.b, bts.testDB, bts.mockCtrl),
		imageCVEDatastore,
		resolvers.CreateTestImageComponentCVEEdgeDatastore(bts.b, bts.testDB),
		resolvers.CreateTestImageCVEEdgeDatastore(bts.b, bts.testDB),
		resolvers.CreateTestDeploymentDatastore(bts.b, bts.testDB, bts.mockCtrl, imageDataStore),
	)

	var err error
	collectionStore := collectionPostgres.CreateTableAndNewStore(bts.ctx, bts.testDB.DB, bts.testDB.GetGormDB(bts.b))
	index := collectionPostgres.NewIndexer(bts.testDB.DB)
	_, bts.collectionQueryResolver, err = collectionDS.New(collectionStore, collectionSearch.New(collectionStore, index))
	require.NoError(bts.b, err)

	bts.watchedImageDatastore = watchedImageDS.GetTestPostgresDataStore(bts.b, bts.testDB.DB)
	bts.clusterDatastore = clusterDSMocks.NewMockDataStore(bts.mockCtrl)
	bts.namespaceDatastore = namespaceDSMocks.NewMockDataStore(bts.mockCtrl)

	bts.reportGenerator = newReportGeneratorImpl(bts.testDB, nil, bts.resolver.DeploymentDataStore,
		bts.watchedImageDatastore, bts.collectionQueryResolver, nil, nil, bts.clusterDatastore,
		bts.namespaceDatastore, imageCVEDatastore, bts.schema)
}

func (bts *ReportGeneratorBenchmarkTestSuite) teardownTestSuite() {
	bts.testDB.Teardown(bts.b)
}

func (bts *ReportGeneratorBenchmarkTestSuite) upsertManyImages(images []*storage.Image) {
	for _, img := range images {
		err := bts.resolver.ImageDataStore.UpsertImage(bts.ctx, img)
		require.NoError(bts.b, err)
	}
}

func (bts *ReportGeneratorBenchmarkTestSuite) upsertManyWatchedImages(images []*storage.Image) {
	for _, img := range images {
		err := bts.watchedImageDatastore.UpsertWatchedImage(bts.ctx, img.Name.FullName)
		require.NoError(bts.b, err)
	}
}

func (bts *ReportGeneratorBenchmarkTestSuite) upsertManyDeployments(deployments []*storage.Deployment) {
	for _, dep := range deployments {
		err := bts.resolver.DeploymentDataStore.UpsertDeployment(bts.ctx, dep)
		require.NoError(bts.b, err)
	}
}
