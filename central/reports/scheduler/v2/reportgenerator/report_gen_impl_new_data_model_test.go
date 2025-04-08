//go:build sql_integration

package reportgenerator

import (
	"context"
	"fmt"
	"testing"
	"time"

	blobDS "github.com/stackrox/rox/central/blob/datastore"
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
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type vulnReportDataNewDataModel struct {
	deploymentNames []string
	imageNames      []string
	componentNames  []string
	cveNames        []string
	cvss            []float64
}

func TestVulnReportingNewDataModel(t *testing.T) {
	suite.Run(t, new(NewDataModelEnhancedReportingTestSuite))
}

type NewDataModelEnhancedReportingTestSuite struct {
	suite.Suite

	ctx                   context.Context
	testDB                *pgtest.TestPostgres
	watchedImageDatastore watchedImageDS.DataStore
	clusterDatastore      *clusterDSMocks.MockDataStore
	namespaceDatastore    *namespaceDSMocks.MockDataStore
	reportGenerator       *reportGeneratorImpl
}

func (s *NewDataModelEnhancedReportingTestSuite) TearDownTest() {
	s.truncateTable(postgresSchema.DeploymentsTableName)
	s.truncateTable(postgresSchema.ImagesTableName)
	s.truncateTable(postgresSchema.ImageComponentV2TableName)
	s.truncateTable(postgresSchema.ImageCvesV2TableName)
	s.truncateTable(postgresSchema.CollectionsTableName)
	// os.Setenv("ROX_FLATTEN_CVE_DATA", "false")
}

func (s *NewDataModelEnhancedReportingTestSuite) SetupSuite() {

	// os.Setenv("ROX_FLATTEN_CVE_DATA", "true")
	if !features.FlattenCVEData.Enabled() {
		s.T().Skip()
	}

	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = pgtest.ForT(s.T())

	// set up tables
	imageDataStore := resolvers.CreateTestImageV2Datastore(s.T(), s.testDB, mockCtrl)
	s.watchedImageDatastore = watchedImageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	resolver, schema := resolvers.SetupTestResolver(s.T(),
		imagesView.NewImageView(s.testDB.DB),
		imageDataStore,
		resolvers.CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl),
		resolvers.CreateTestImageCVEV2Datastore(s.T(), s.testDB),
		resolvers.CreateTestDeploymentDatastore(s.T(), s.testDB, mockCtrl, imageDataStore),
	)
	collectionStore := collectionPostgres.CreateTableAndNewStore(s.ctx, s.testDB.DB, s.testDB.GetGormDB(s.T()))
	_, collectionQueryResolver, err := collectionDS.New(collectionStore, collectionSearch.New(collectionStore))
	s.NoError(err)
	s.clusterDatastore = clusterDSMocks.NewMockDataStore(mockCtrl)
	s.namespaceDatastore = namespaceDSMocks.NewMockDataStore(mockCtrl)

	// Add Test Data to DataStores
	clusters := []*storage.Cluster{
		{Id: uuid.NewV4().String(), Name: "c1"},
		{Id: uuid.NewV4().String(), Name: "c2"},
	}

	namespaces := testNamespaces(clusters, 2)
	deployments, images := testDeploymentsWithImages(namespaces, 1)
	// insert deployments in deployment table
	for _, dep := range deployments {
		err := resolver.DeploymentDataStore.UpsertDeployment(s.ctx, dep)
		s.NoError(err)
	}
	// upsert deployed image in image table
	for _, image := range images {
		err := resolver.ImageDataStore.UpsertImage(s.ctx, image)
		s.NoError(err)
	}

	// upsert watched images
	watchedImages := testWatchedImages(2)
	for _, image := range watchedImages {
		err := resolver.ImageDataStore.UpsertImage(s.ctx, image)
		s.NoError(err)
	}
	s.upsertManyWatchedImages(watchedImages)

	s.clusterDatastore.EXPECT().GetClusters(gomock.Any()).
		Return(clusters, nil).AnyTimes()

	s.namespaceDatastore.EXPECT().GetAllNamespaces(gomock.Any()).
		Return(namespaces, nil).AnyTimes()

	blobStore := blobDS.NewTestDatastore(s.T(), s.testDB.DB)

	s.reportGenerator = newReportGeneratorImpl(s.testDB, nil, resolver.DeploymentDataStore,
		s.watchedImageDatastore, collectionQueryResolver, nil, blobStore, s.clusterDatastore,
		s.namespaceDatastore, resolver.ImageCVEDataStore, resolver.ImageCVEV2DataStore, schema)
}
func (s *NewDataModelEnhancedReportingTestSuite) upsertManyWatchedImages(images []*storage.Image) {
	for _, img := range images {
		err := s.watchedImageDatastore.UpsertWatchedImage(s.ctx, img.Name.FullName)
		s.NoError(err)
	}
}

func (s *NewDataModelEnhancedReportingTestSuite) truncateTable(name string) {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", name)
	_, err := s.testDB.Exec(s.ctx, sql)
	s.NoError(err)
}

func (s *NewDataModelEnhancedReportingTestSuite) TestGetReportData() {

	testCases := []struct {
		name       string
		collection *storage.ResourceCollection
		fixability storage.VulnerabilityReportFilters_Fixability
		severities []storage.VulnerabilitySeverity
		imageTypes []storage.VulnerabilityReportFilters_ImageType
		scopeRules []*storage.SimpleAccessScope_Rules
		expected   *vulnReportDataNewDataModel
	}{
		{
			name:       "Include all deployments; CVEs with both fixabilities and all severities; Nil scope rules",
			collection: testCollection("col1", "", "", ""),
			fixability: storage.VulnerabilityReportFilters_BOTH,
			severities: allSeverities(),
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED},
			scopeRules: nil,
			expected: &vulnReportDataNewDataModel{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp", "CVE-nonFixable_low-c1_ns2_dep0_img_comp",
					"CVE-fixable_critical-c2_ns1_dep0_img_comp", "CVE-nonFixable_low-c2_ns1_dep0_img_comp",
					"CVE-fixable_critical-c2_ns2_dep0_img_comp", "CVE-nonFixable_low-c2_ns2_dep0_img_comp",
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			reportSnap := testReportSnapshot(tc.collection.GetId(), tc.fixability, tc.severities, tc.imageTypes, tc.scopeRules)
			// Test get data using SQF
			reportData, err := s.reportGenerator.getReportDataSQF(reportSnap, tc.collection, time.Time{})
			s.NoError(err)
			collected := collectVulnReportDataSQFNewDataModel(reportData.CVEResponses)
			s.ElementsMatch(tc.expected.deploymentNames, collected.deploymentNames)
			s.ElementsMatch(tc.expected.imageNames, collected.imageNames)
			s.ElementsMatch(tc.expected.componentNames, collected.componentNames)
			s.ElementsMatch(tc.expected.cveNames, collected.cveNames)
			s.Equal(len(tc.expected.cveNames), reportData.NumDeployedImageResults+reportData.NumWatchedImageResults)
			s.Equal(len(tc.expected.cveNames), len(collected.cvss))
		})
	}

}

func collectVulnReportDataSQFNewDataModel(cveResponses []*ImageCVEQueryResponse) *vulnReportDataNewDataModel {
	deploymentNames := set.NewStringSet()
	imageNames := set.NewStringSet()
	componentNames := set.NewStringSet()
	cveNames := make([]string, 0)
	cvss := make([]float64, 0)

	for _, res := range cveResponses {
		if res.GetDeployment() != "" {
			deploymentNames.Add(res.GetDeployment())
		}
		imageNames.Add(res.GetImage())
		componentNames.Add(res.GetComponent())
		cveNames = append(cveNames, res.GetCVE())
		cvss = append(cvss, res.GetCVSS())
	}
	return &vulnReportDataNewDataModel{
		deploymentNames: deploymentNames.AsSlice(),
		imageNames:      imageNames.AsSlice(),
		componentNames:  componentNames.AsSlice(),
		cveNames:        cveNames,
		cvss:            cvss,
	}
}
