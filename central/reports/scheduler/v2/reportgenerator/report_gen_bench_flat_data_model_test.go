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
	bts.watchedImageDatastore = watchedImageDS.GetTestPostgresDataStore(bts.b, bts.testDB.DB)
	var schema *graphql.Schema
	// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
	if features.FlattenImageData.Enabled() {
		imgV2DataStore := resolvers.CreateTestImageV2Datastore(bts.b, bts.testDB, mockCtrl)
		bts.resolver, schema = resolvers.SetupTestResolver(bts.b,
			imagesView.NewImageView(bts.testDB.DB),
			imgV2DataStore,
			resolvers.CreateTestImageComponentV2Datastore(bts.b, bts.testDB, mockCtrl),
			resolvers.CreateTestImageCVEV2Datastore(bts.b, bts.testDB),
			resolvers.CreateTestDeploymentDatastoreWithImageV2(bts.b, bts.testDB, mockCtrl, imgV2DataStore),
			deploymentsView.NewDeploymentView(bts.testDB.DB),
		)
	} else {
		imageDataStore := resolvers.CreateTestImageDatastore(bts.b, bts.testDB, mockCtrl)
		bts.resolver, schema = resolvers.SetupTestResolver(bts.b,
			imagesView.NewImageView(bts.testDB.DB),
			imageDataStore,
			resolvers.CreateTestImageComponentV2Datastore(bts.b, bts.testDB, mockCtrl),
			resolvers.CreateTestImageCVEV2Datastore(bts.b, bts.testDB),
			resolvers.CreateTestDeploymentDatastore(bts.b, bts.testDB, mockCtrl, imageDataStore),
			deploymentsView.NewDeploymentView(bts.testDB.DB),
		)
	}
	collectionStore := collectionPostgres.New(bts.testDB)
	_, collectionQueryResolver, err := collectionDS.New(collectionStore)
	require.NoError(bts.b, err)
	bts.clusterDatastore = clusterDSMocks.NewMockDataStore(mockCtrl)
	bts.namespaceDatastore = namespaceDSMocks.NewMockDataStore(mockCtrl)

	bts.reportGenerator = newReportGeneratorImpl(bts.testDB, nil, bts.resolver.DeploymentDataStore,
		bts.watchedImageDatastore, collectionQueryResolver, nil, nil, bts.clusterDatastore,
		bts.namespaceDatastore, bts.resolver.ImageCVEV2DataStore, schema)
}

func BenchmarkEntityScopeReportGenerator(b *testing.B) {
	ctx := loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(b)
	testDB := pgtest.ForT(b)

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

	clusterDatastore := clusterDSMocks.NewMockDataStore(mockCtrl)
	nsDatastore, err := namespaceDS.GetTestPostgresDataStore(b, testDB.DB)
	require.NoError(b, err)

	reportGenerator := newReportGeneratorImpl(testDB, nil, resolver.DeploymentDataStore,
		watchedImageDatastore, nil, nil, nil, clusterDatastore,
		nsDatastore, resolver.ImageCVEV2DataStore, schema)

	clusters := []*storage.Cluster{
		{Id: uuid.NewV4().String(), Name: "c1"},
		{Id: uuid.NewV4().String(), Name: "c2"},
	}

	namespaces := testNamespaces(clusters, 10)

	// Add labels to first 5 namespaces of each cluster (ns1-ns5 get team=backend)
	for i, ns := range namespaces {
		if i%10 < 5 {
			ns.Labels = map[string]string{"team": "backend"}
		} else {
			ns.Labels = map[string]string{"team": "frontend"}
		}
		err := nsDatastore.AddNamespace(ctx, ns)
		require.NoError(b, err)
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

	clusterDatastore.EXPECT().GetClusters(gomock.Any()).
		Return(clusters, nil).AnyTimes()

	allImageTypes := []storage.VulnerabilityReportFilters_ImageType{
		storage.VulnerabilityReportFilters_DEPLOYED,
		storage.VulnerabilityReportFilters_WATCHED,
	}
	sinceStartDate := time.Now().AddDate(-1, 0, 0)

	// c1: 10 namespaces × 100 deployments = 1000 deployed images (2 CVEs each)
	// c2: 10 namespaces × 100 deployments = 1000 deployed images (2 CVEs each)
	// ns1-ns5 per cluster have label team=backend (5 ns × 100 deps = 500 per cluster)
	// 500 watched images (2 CVEs each): registry=quay.io, label=app:watch
	// Each image has: 1 fixable_critical CVE (CVSS=9.0, EPSS=0.7) + 1 nonFixable_low CVE (CVSS=2.0, EPSS=0.1)

	b.Run("EntityScope_ClusterScope_WithNamespaceLabel", func(b *testing.B) {
		entityScope := &storage.EntityScope{
			Rules: []*storage.EntityScopeRule{
				{
					Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
					Field:  storage.EntityField_FIELD_NAME,
					Values: []*storage.RuleValue{
						{Value: "c1", MatchType: storage.MatchType_EXACT},
					},
				},
				{
					Entity: storage.EntityType_ENTITY_TYPE_NAMESPACE,
					Field:  storage.EntityField_FIELD_LABEL,
					Values: []*storage.RuleValue{
						{Value: "team=backend", MatchType: storage.MatchType_EXACT},
					},
				},
			},
		}
		scopeRules := []*storage.SimpleAccessScope_Rules{
			{IncludedClusters: []string{"c1"}},
		}
		// CVSS>=7.0 matches fixable_critical only
		// Entity scope: cluster=c1 AND namespace label team=backend
		// Deployed: c1 ns1-ns5 = 500 images × 1 CVE = 500; Watched: 500 × 1 CVE = 500
		expectedRowCount := 1000
		reportSnap := testEntityScopeReportSnapshot(entityScope, "CVSS:>=7.0", allImageTypes, scopeRules)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			reportData, err := reportGenerator.getReportDataSQF(reportSnap, nil, sinceStartDate)
			require.NoError(b, err)
			require.Equal(b, expectedRowCount, len(reportData.CVEResponses))
		}
	})

	b.Run("EntityScope_NamespaceScope_WithEPSSFilter", func(b *testing.B) {
		entityScope := &storage.EntityScope{
			Rules: []*storage.EntityScopeRule{
				{
					Entity: storage.EntityType_ENTITY_TYPE_NAMESPACE,
					Field:  storage.EntityField_FIELD_NAME,
					Values: []*storage.RuleValue{
						{Value: "ns1", MatchType: storage.MatchType_EXACT},
					},
				},
			},
		}
		scopeRules := []*storage.SimpleAccessScope_Rules{
			{IncludedClusters: []string{"c1", "c2"}},
		}
		// EPSS>=0.5 matches fixable_critical only (EPSS=0.7)
		// Deployed: ns1 across 2 clusters = 200 images × 1 CVE = 200; Watched: 500 × 1 CVE = 500
		expectedRowCount := 700
		reportSnap := testEntityScopeReportSnapshot(entityScope, "EPSS Probability:>=0.5", allImageTypes, scopeRules)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			reportData, err := reportGenerator.getReportDataSQF(reportSnap, nil, sinceStartDate)
			require.NoError(b, err)
			require.Equal(b, expectedRowCount, len(reportData.CVEResponses))
		}
	})

	b.Run("EntityScope_DeploymentRegex_WithCVSSFilter", func(b *testing.B) {
		entityScope := &storage.EntityScope{
			Rules: []*storage.EntityScopeRule{
				{
					Entity: storage.EntityType_ENTITY_TYPE_DEPLOYMENT,
					Field:  storage.EntityField_FIELD_NAME,
					Values: []*storage.RuleValue{
						{Value: "c1_.*", MatchType: storage.MatchType_REGEX},
					},
				},
			},
		}
		scopeRules := []*storage.SimpleAccessScope_Rules{
			{IncludedClusters: []string{"c1"}},
		}
		// CVSS>=7.0 matches fixable_critical only
		// Deployed: 1000 c1 deployments matching c1_.* × 1 CVE = 1000; Watched: 500 × 1 CVE = 500
		expectedRowCount := 1500
		reportSnap := testEntityScopeReportSnapshot(entityScope, "CVSS:>=7.0", allImageTypes, scopeRules)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			reportData, err := reportGenerator.getReportDataSQF(reportSnap, nil, sinceStartDate)
			require.NoError(b, err)
			require.Equal(b, expectedRowCount, len(reportData.CVEResponses))
		}
	})
}

func (bts *FlatDataModelReportGeneratorBenchmarkTestSuite) upsertManyImages(images []*storage.Image) {
	// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
	if features.FlattenImageData.Enabled() {
		for _, img := range images {
			err := bts.resolver.ImageV2DataStore.UpsertImage(bts.ctx, imageUtils.ConvertToV2(img))
			require.NoError(bts.b, err)
		}
	} else {
		for _, img := range images {
			err := bts.resolver.ImageDataStore.UpsertImage(bts.ctx, img)
			require.NoError(bts.b, err)
		}
	}
}

func (bts *FlatDataModelReportGeneratorBenchmarkTestSuite) upsertManyWatchedImages(images []*storage.Image) {
	for _, img := range images {
		err := bts.watchedImageDatastore.UpsertWatchedImage(bts.ctx, img.GetName().GetFullName())
		require.NoError(bts.b, err)
	}
}

func (bts *FlatDataModelReportGeneratorBenchmarkTestSuite) upsertManyDeployments(deployments []*storage.Deployment) {
	for _, dep := range deployments {
		err := bts.resolver.DeploymentDataStore.UpsertDeployment(bts.ctx, dep)
		require.NoError(bts.b, err)
	}
}
