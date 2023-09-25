package reportgenerator

import (
	"context"
	"fmt"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	namespaceDSMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/central/reports/common"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	collectionSearch "github.com/stackrox/rox/central/resourcecollection/datastore/search"
	collectionPostgres "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	types2 "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
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
	b.Setenv(env.VulnReportingEnhancements.EnvVar(), "true")
	if !env.VulnReportingEnhancements.BooleanSetting() {
		b.Skip("Skip tests when ROX_VULN_MGMT_REPORTING_ENHANCEMENTS disabled")
		b.SkipNow()
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
			cveResponses, err := bts.reportGenerator.getReportDataSQF(reportSnap, collection, nil)
			require.NoError(b, err)
			require.Equal(b, expectedRowCount, len(cveResponses))
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

func testNamespaces(clusters []*storage.Cluster, namespacesPerCluster int) []*storage.NamespaceMetadata {
	namespaces := make([]*storage.NamespaceMetadata, 0)
	for _, cluster := range clusters {
		for i := 0; i < namespacesPerCluster; i++ {
			namespaceName := fmt.Sprintf("ns%d", i+1)
			namespaces = append(namespaces, &storage.NamespaceMetadata{
				Id:          uuid.NewV4().String(),
				Name:        namespaceName,
				ClusterId:   cluster.Id,
				ClusterName: cluster.Name,
			})
		}
	}
	return namespaces
}

func allSeverities() []storage.VulnerabilitySeverity {
	return []storage.VulnerabilitySeverity{
		storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
		storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
		storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
	}
}

func testDeploymentsWithImages(namespaces []*storage.NamespaceMetadata, numDeploymentsPerNamespace int) ([]*storage.Deployment, []*storage.Image) {
	capacity := len(namespaces) * numDeploymentsPerNamespace
	deployments := make([]*storage.Deployment, 0, capacity)
	images := make([]*storage.Image, 0, capacity)

	for _, namespace := range namespaces {
		for i := 0; i < numDeploymentsPerNamespace; i++ {
			depName := fmt.Sprintf("%s_%s_dep%d", namespace.ClusterName, namespace.Name, i)
			image := testImage(depName)
			deployment := testDeployment(depName, namespace, image)
			deployments = append(deployments, deployment)
			images = append(images, image)
		}
	}
	return deployments, images
}

func testDeployment(deploymentName string, namespace *storage.NamespaceMetadata, image *storage.Image) *storage.Deployment {
	return &storage.Deployment{
		Name:        deploymentName,
		Id:          uuid.NewV4().String(),
		ClusterName: namespace.ClusterName,
		ClusterId:   namespace.ClusterId,
		Namespace:   namespace.Name,
		NamespaceId: namespace.Id,
		Containers: []*storage.Container{
			{
				Name:  fmt.Sprintf("%s_container", deploymentName),
				Image: types2.ToContainerImage(image),
			},
		},
	}
}

func testWatchedImages(numImages int) []*storage.Image {
	images := make([]*storage.Image, 0, numImages)
	for i := 0; i < numImages; i++ {
		imgNamePrefix := fmt.Sprintf("w%d", i)
		image := testImage(imgNamePrefix)
		images = append(images, image)
	}
	return images
}

func testImage(prefix string) *storage.Image {
	t, err := ptypes.TimestampProto(time.Unix(0, 1000))
	utils.CrashOnError(err)
	return &storage.Image{
		Id:   fmt.Sprintf("%s_img", prefix),
		Name: &storage.ImageName{FullName: fmt.Sprintf("%s_img", prefix)},
		SetComponents: &storage.Image_Components{
			Components: 1,
		},
		SetCves: &storage.Image_Cves{
			Cves: 2,
		},
		Scan: &storage.ImageScan{
			ScanTime: t,
			Components: []*storage.EmbeddedImageScanComponent{
				{
					Name:    fmt.Sprintf("%s_img_comp", prefix),
					Version: "1.0",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve: fmt.Sprintf("CVE-fixable_critical-%s_img_comp", prefix),
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "1.1",
							},
							Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							Link:     "link",
						},
						{
							Cve:      fmt.Sprintf("CVE-nonFixable_low-%s_img_comp", prefix),
							Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
							Link:     "link",
						},
					},
				},
			},
		},
	}
}

func testCollection(collectionName, cluster, namespace, deployment string) *storage.ResourceCollection {
	collection := &storage.ResourceCollection{
		Name: collectionName,
		ResourceSelectors: []*storage.ResourceSelector{
			{
				Rules: []*storage.SelectorRule{},
			},
		},
	}
	if cluster != "" {
		collection.ResourceSelectors[0].Rules = append(collection.ResourceSelectors[0].Rules, &storage.SelectorRule{
			FieldName: pkgSearch.Cluster.String(),
			Operator:  storage.BooleanOperator_OR,
			Values: []*storage.RuleValue{
				{
					Value:     cluster,
					MatchType: storage.MatchType_EXACT,
				},
			},
		})
	}
	if namespace != "" {
		collection.ResourceSelectors[0].Rules = append(collection.ResourceSelectors[0].Rules, &storage.SelectorRule{
			FieldName: pkgSearch.Namespace.String(),
			Operator:  storage.BooleanOperator_OR,
			Values: []*storage.RuleValue{
				{
					Value:     namespace,
					MatchType: storage.MatchType_EXACT,
				},
			},
		})
	}
	var deploymentVal string
	var matchType storage.MatchType
	if deployment != "" {
		deploymentVal = deployment
		matchType = storage.MatchType_EXACT
	} else {
		deploymentVal = ".*"
		matchType = storage.MatchType_REGEX
	}
	collection.ResourceSelectors[0].Rules = append(collection.ResourceSelectors[0].Rules, &storage.SelectorRule{
		FieldName: pkgSearch.DeploymentName.String(),
		Operator:  storage.BooleanOperator_OR,
		Values: []*storage.RuleValue{
			{
				Value:     deploymentVal,
				MatchType: matchType,
			},
		},
	})

	return collection
}

func testReportSnapshot(collectionID string,
	fixability storage.VulnerabilityReportFilters_Fixability,
	severities []storage.VulnerabilitySeverity,
	imageTypes []storage.VulnerabilityReportFilters_ImageType,
	scopeRules []*storage.SimpleAccessScope_Rules) *storage.ReportSnapshot {
	snap := fixtures.GetReportSnapshot()
	snap.Filter = &storage.ReportSnapshot_VulnReportFilters{
		VulnReportFilters: &storage.VulnerabilityReportFilters{
			Fixability: fixability,
			Severities: severities,
			ImageTypes: imageTypes,
			CvesSince: &storage.VulnerabilityReportFilters_AllVuln{
				AllVuln: true,
			},
			AccessScopeRules: scopeRules,
		},
	}
	snap.Collection = &storage.CollectionSnapshot{
		Id:   collectionID,
		Name: collectionID,
	}
	return snap
}

func countNumRows(deployedImgResults []common.DeployedImagesResult, watchedImgResults []common.WatchedImagesResult) int {
	count := 0
	for _, res := range deployedImgResults {
		for _, dep := range res.Deployments {
			for _, img := range dep.Images {
				for _, comp := range img.ImageComponents {
					count += len(comp.ImageVulnerabilities)
				}
			}
		}
	}

	for _, res := range watchedImgResults {
		for _, img := range res.Images {
			for _, comp := range img.ImageComponents {
				count += len(comp.ImageVulnerabilities)
			}
		}
	}

	return count
}
