//go:build sql_integration

package reportgenerator

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	namespaceDSMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/central/reports/common"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	collectionSearch "github.com/stackrox/rox/central/resourcecollection/datastore/search"
	collectionPostgres "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
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

func TestEnhancedReporting(t *testing.T) {
	if features.FlattenCVEData.Enabled() {
		t.Skip()
	}
	suite.Run(t, new(EnhancedReportingTestSuite))
}

type EnhancedReportingTestSuite struct {
	suite.Suite
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

	blobStore blobDS.Datastore
}

type vulnReportData struct {
	deploymentNames []string
	imageNames      []string
	componentNames  []string
	cveNames        []string
	cvss            []float64
}

func (s *EnhancedReportingTestSuite) SetupSuite() {
	if features.FlattenCVEData.Enabled() {
		s.T().Skip()
	}
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	s.mockCtrl = gomock.NewController(s.T())
	s.testDB = resolvers.SetupTestPostgresConn(s.T())
	imageDataStore := resolvers.CreateTestImageDatastore(s.T(), s.testDB, s.mockCtrl)
	imageCVEDatastore := resolvers.CreateTestImageCVEDatastore(s.T(), s.testDB)
	s.resolver, s.schema = resolvers.SetupTestResolver(s.T(),
		imageDataStore,
		imagesView.NewImageView(s.testDB.DB),
		resolvers.CreateTestImageComponentDatastore(s.T(), s.testDB, s.mockCtrl),
		imageCVEDatastore,
		resolvers.CreateTestImageComponentCVEEdgeDatastore(s.T(), s.testDB),
		resolvers.CreateTestImageCVEEdgeDatastore(s.T(), s.testDB),
		resolvers.CreateTestDeploymentDatastore(s.T(), s.testDB, s.mockCtrl, imageDataStore),
		deploymentsView.NewDeploymentView(s.testDB.DB),
	)

	var err error
	collectionStore := collectionPostgres.CreateTableAndNewStore(s.ctx, s.testDB.DB, s.testDB.GetGormDB(s.T()))
	_, s.collectionQueryResolver, err = collectionDS.New(collectionStore, collectionSearch.New(collectionStore))
	s.NoError(err)

	s.watchedImageDatastore = watchedImageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.clusterDatastore = clusterDSMocks.NewMockDataStore(s.mockCtrl)
	s.namespaceDatastore = namespaceDSMocks.NewMockDataStore(s.mockCtrl)

	s.blobStore = blobDS.NewTestDatastore(s.T(), s.testDB.DB)
	s.reportGenerator = newReportGeneratorImpl(s.testDB, nil, s.resolver.DeploymentDataStore,
		s.watchedImageDatastore, s.collectionQueryResolver, nil, s.blobStore, s.clusterDatastore,
		s.namespaceDatastore, imageCVEDatastore, s.resolver.ImageCVEV2DataStore, s.schema)
}

func (s *EnhancedReportingTestSuite) TearDownTest() {
	s.truncateTable(postgresSchema.DeploymentsTableName)
	s.truncateTable(postgresSchema.ImagesTableName)
	s.truncateTable(postgresSchema.ImageComponentsTableName)
	s.truncateTable(postgresSchema.ImageCvesTableName)
	s.truncateTable(postgresSchema.CollectionsTableName)
}

func (s *EnhancedReportingTestSuite) TestSaveReportData() {
	configID := "configid"
	data := []byte("something something")
	buf := bytes.NewBuffer(data)

	// Save report
	reportID := "reportid"
	s.Require().NoError(s.reportGenerator.saveReportData(configID, reportID, buf))
	newBuf, _, exists, err := s.blobStore.GetBlobWithDataInBuffer(s.ctx, common.GetReportBlobPath(configID, reportID))
	s.Require().NoError(err)
	s.Require().True(exists)
	s.Equal(data, newBuf.Bytes())

	// Save empty report
	reportID = "anotherid"
	s.Require().Error(s.reportGenerator.saveReportData(configID, reportID, nil))
}

func (s *EnhancedReportingTestSuite) TestGetReportData() {
	clusters := []*storage.Cluster{
		{Id: uuid.NewV4().String(), Name: "c1"},
		{Id: uuid.NewV4().String(), Name: "c2"},
	}

	namespaces := testNamespaces(clusters, 2)

	deployments, images := testDeploymentsWithImages(namespaces, 1)
	s.upsertManyImages(images)
	s.upsertManyDeployments(deployments)

	watchedImages := testWatchedImages(2)
	s.upsertManyImages(watchedImages)
	s.upsertManyWatchedImages(watchedImages)

	s.clusterDatastore.EXPECT().GetClusters(gomock.Any()).
		Return(clusters, nil).AnyTimes()

	s.namespaceDatastore.EXPECT().GetAllNamespaces(gomock.Any()).
		Return(namespaces, nil).AnyTimes()

	testCases := []struct {
		name       string
		collection *storage.ResourceCollection
		fixability storage.VulnerabilityReportFilters_Fixability
		severities []storage.VulnerabilitySeverity
		imageTypes []storage.VulnerabilityReportFilters_ImageType
		scopeRules []*storage.SimpleAccessScope_Rules
		expected   *vulnReportData
	}{
		{
			name:       "Include all deployments; CVEs with both fixabilities and all severities; Nil scope rules",
			collection: testCollection("col1", "", "", ""),
			fixability: storage.VulnerabilityReportFilters_BOTH,
			severities: allSeverities(),
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED},
			scopeRules: nil,
			expected: &vulnReportData{
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
		{
			name:       "Include all deployments; Fixable CVEs with CRITICAL severity; Nil scope rules",
			collection: testCollection("col2", "", "", ""),
			fixability: storage.VulnerabilityReportFilters_FIXABLE,
			severities: []storage.VulnerabilitySeverity{
				storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			},
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED},
			scopeRules: nil,
			expected: &vulnReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp",
					"CVE-fixable_critical-c2_ns1_dep0_img_comp",
					"CVE-fixable_critical-c2_ns2_dep0_img_comp",
				},
			},
		},
		{
			name:       "Include deployments from cluster c1 and namespace ns1; CVEs with both fixabilities and all severities; Nil scope rules",
			collection: testCollection("col3", "c1", "ns1", ""),
			fixability: storage.VulnerabilityReportFilters_BOTH,
			severities: allSeverities(),
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED},
			scopeRules: nil,
			expected: &vulnReportData{
				deploymentNames: []string{"c1_ns1_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp",
				},
			},
		},
		{
			name:       "Include all deployments + watched images; CVEs with both fixabilities and all severities; Nil scope rules",
			collection: testCollection("col4", "", "", ""),
			fixability: storage.VulnerabilityReportFilters_BOTH,
			severities: allSeverities(),
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{
				storage.VulnerabilityReportFilters_DEPLOYED,
				storage.VulnerabilityReportFilters_WATCHED,
			},
			expected: &vulnReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames: []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp",
					"w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp", "CVE-nonFixable_low-c1_ns2_dep0_img_comp",
					"CVE-fixable_critical-c2_ns1_dep0_img_comp", "CVE-nonFixable_low-c2_ns1_dep0_img_comp",
					"CVE-fixable_critical-c2_ns2_dep0_img_comp", "CVE-nonFixable_low-c2_ns2_dep0_img_comp",
					"CVE-fixable_critical-w0_img_comp", "CVE-nonFixable_low-w0_img_comp",
					"CVE-fixable_critical-w1_img_comp", "CVE-nonFixable_low-w1_img_comp",
				},
			},
		},
		{
			name:       "Include watched images only; Fixable CVEs with CRITICAL severity; Nil scope rules",
			collection: testCollection("col5", "", "", ""),
			fixability: storage.VulnerabilityReportFilters_FIXABLE,
			severities: []storage.VulnerabilitySeverity{
				storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			},
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_WATCHED},
			scopeRules: nil,
			expected: &vulnReportData{
				deploymentNames: []string{},
				imageNames:      []string{"w0_img", "w1_img"},
				componentNames:  []string{"w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-w0_img_comp",
					"CVE-fixable_critical-w1_img_comp",
				},
			},
		},
		{
			name:       "Include all deployments + all CVEs; Empty scope rules",
			collection: testCollection("col6", "", "", ""),
			fixability: storage.VulnerabilityReportFilters_BOTH,
			severities: allSeverities(),
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{
				storage.VulnerabilityReportFilters_DEPLOYED,
				storage.VulnerabilityReportFilters_WATCHED,
			},
			scopeRules: make([]*storage.SimpleAccessScope_Rules, 0),
			expected: &vulnReportData{
				deploymentNames: []string{},
				imageNames:      []string{"w0_img", "w1_img"},
				componentNames:  []string{"w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-w0_img_comp", "CVE-nonFixable_low-w0_img_comp",
					"CVE-fixable_critical-w1_img_comp", "CVE-nonFixable_low-w1_img_comp",
				},
			},
		},
		{
			name:       "Include all deployments + all CVEs; Non-empty scope rules",
			collection: testCollection("col7", "", "", ""),
			fixability: storage.VulnerabilityReportFilters_BOTH,
			severities: allSeverities(),
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{
				storage.VulnerabilityReportFilters_DEPLOYED,
			},
			scopeRules: []*storage.SimpleAccessScope_Rules{
				{
					IncludedClusters: []string{"c1"},
				},
				{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{ClusterName: "c2", NamespaceName: "ns1"},
					},
				},
			},
			expected: &vulnReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp", "CVE-nonFixable_low-c1_ns2_dep0_img_comp",
					"CVE-fixable_critical-c2_ns1_dep0_img_comp", "CVE-nonFixable_low-c2_ns1_dep0_img_comp",
				},
			},
		},
		{
			name:       "Collection matching all deps from cluster c1; Scope allowing cluster c1 and namespace ns1",
			collection: testCollection("col8", "c1", "", ""),
			fixability: storage.VulnerabilityReportFilters_BOTH,
			severities: allSeverities(),
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{
				storage.VulnerabilityReportFilters_DEPLOYED,
			},
			scopeRules: []*storage.SimpleAccessScope_Rules{
				{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{ClusterName: "c1", NamespaceName: "ns1"},
					},
				},
			},
			expected: &vulnReportData{
				deploymentNames: []string{"c1_ns1_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp",
				},
			},
		},
		{
			name:       "Collection matching cluster c1; Scope allowing cluster c2",
			collection: testCollection("col9", "c1", "", ""),
			fixability: storage.VulnerabilityReportFilters_BOTH,
			severities: allSeverities(),
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{
				storage.VulnerabilityReportFilters_DEPLOYED,
			},
			scopeRules: []*storage.SimpleAccessScope_Rules{
				{
					IncludedClusters: []string{"c2"},
				},
			},
			expected: &vulnReportData{
				deploymentNames: []string{},
				imageNames:      []string{},
				componentNames:  []string{},
				cveNames:        []string{},
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			reportSnap := testReportSnapshot(tc.collection.GetId(), tc.fixability, tc.severities, tc.imageTypes, tc.scopeRules)
			// Test get data using SQF
			reportData, err := s.reportGenerator.getReportDataSQF(reportSnap, tc.collection, time.Time{})
			s.NoError(err)
			collected := collectVulnReportDataSQF(reportData.CVEResponses)
			s.ElementsMatch(tc.expected.deploymentNames, collected.deploymentNames)
			s.ElementsMatch(tc.expected.imageNames, collected.imageNames)
			s.ElementsMatch(tc.expected.componentNames, collected.componentNames)
			s.ElementsMatch(tc.expected.cveNames, collected.cveNames)
			s.Equal(len(tc.expected.cveNames), reportData.NumDeployedImageResults+reportData.NumWatchedImageResults)
			s.Equal(len(tc.expected.cveNames), len(collected.cvss))
		})
	}
}

func (s *EnhancedReportingTestSuite) truncateTable(name string) {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", name)
	_, err := s.testDB.Exec(s.ctx, sql)
	s.NoError(err)
}

func (s *EnhancedReportingTestSuite) upsertManyImages(images []*storage.Image) {
	for _, img := range images {
		err := s.resolver.ImageDataStore.UpsertImage(s.ctx, img)
		s.NoError(err)
	}
}

func (s *EnhancedReportingTestSuite) upsertManyWatchedImages(images []*storage.Image) {
	for _, img := range images {
		err := s.watchedImageDatastore.UpsertWatchedImage(s.ctx, img.Name.FullName)
		s.NoError(err)
	}
}

func (s *EnhancedReportingTestSuite) upsertManyDeployments(deployments []*storage.Deployment) {
	for _, dep := range deployments {
		err := s.resolver.DeploymentDataStore.UpsertDeployment(s.ctx, dep)
		s.NoError(err)
	}
}

func collectVulnReportDataSQF(cveResponses []*ImageCVEQueryResponse) *vulnReportData {
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
	return &vulnReportData{
		deploymentNames: deploymentNames.AsSlice(),
		imageNames:      imageNames.AsSlice(),
		componentNames:  componentNames.AsSlice(),
		cveNames:        cveNames,
		cvss:            cvss,
	}
}
