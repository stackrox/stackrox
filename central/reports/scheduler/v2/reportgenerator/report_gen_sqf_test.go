//go:build sql_integration

package reportgenerator

import (
	"context"
	"fmt"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
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
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures"
	types2 "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestEnhancedReportingSQF(t *testing.T) {
	suite.Run(t, new(EnhancedReportingSQFTestSuite))
}

type EnhancedReportingSQFTestSuite struct {
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

func (s *EnhancedReportingSQFTestSuite) SetupSuite() {
	s.T().Setenv(env.VulnReportingEnhancements.EnvVar(), "true")
	if !env.VulnReportingEnhancements.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_VULN_MGMT_REPORTING_ENHANCEMENTS disabled")
		s.T().SkipNow()
	}
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	s.mockCtrl = gomock.NewController(s.T())
	s.testDB = resolvers.SetupTestPostgresConn(s.T())
	imageDataStore := resolvers.CreateTestImageDatastore(s.T(), s.testDB, s.mockCtrl)
	imageCVEDatastore := resolvers.CreateTestImageCVEDatastore(s.T(), s.testDB)
	s.resolver, s.schema = resolvers.SetupTestResolver(s.T(),
		imageDataStore,
		resolvers.CreateTestImageComponentDatastore(s.T(), s.testDB, s.mockCtrl),
		imageCVEDatastore,
		resolvers.CreateTestImageComponentCVEEdgeDatastore(s.T(), s.testDB),
		resolvers.CreateTestImageCVEEdgeDatastore(s.T(), s.testDB),
		resolvers.CreateTestDeploymentDatastore(s.T(), s.testDB, s.mockCtrl, imageDataStore),
	)

	var err error
	collectionStore := collectionPostgres.CreateTableAndNewStore(s.ctx, s.testDB.DB, s.testDB.GetGormDB(s.T()))
	index := collectionPostgres.NewIndexer(s.testDB.DB)
	_, s.collectionQueryResolver, err = collectionDS.New(collectionStore, collectionSearch.New(collectionStore, index))
	s.NoError(err)

	s.watchedImageDatastore = watchedImageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.clusterDatastore = clusterDSMocks.NewMockDataStore(s.mockCtrl)
	s.namespaceDatastore = namespaceDSMocks.NewMockDataStore(s.mockCtrl)

	s.blobStore = blobDS.NewTestDatastore(s.T(), s.testDB.DB)
	s.reportGenerator = newReportGeneratorImpl(s.testDB, nil, s.resolver.DeploymentDataStore,
		s.watchedImageDatastore, s.collectionQueryResolver, nil, s.blobStore, s.clusterDatastore,
		s.namespaceDatastore, imageCVEDatastore, s.schema)
}

func (s *EnhancedReportingSQFTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *EnhancedReportingSQFTestSuite) TearDownTest() {
	s.truncateTable(postgresSchema.DeploymentsTableName)
	s.truncateTable(postgresSchema.ImagesTableName)
	s.truncateTable(postgresSchema.ImageComponentsTableName)
	s.truncateTable(postgresSchema.ImageCvesTableName)
	s.truncateTable(postgresSchema.CollectionsTableName)
}

func (s *EnhancedReportingSQFTestSuite) TestGetReportData() {
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
			startTime := time.Now()
			cveResponses, err := s.reportGenerator.getReportDataSQF(reportSnap, tc.collection, nil)
			sqfDelta := time.Since(startTime).Milliseconds()
			s.NoError(err)
			reportData := extractVulnReportDataSQF(cveResponses)
			s.ElementsMatch(tc.expected.deploymentNames, reportData.deploymentNames)
			s.ElementsMatch(tc.expected.imageNames, reportData.imageNames)
			s.ElementsMatch(tc.expected.componentNames, reportData.componentNames)
			s.ElementsMatch(tc.expected.cveNames, reportData.cveNames)
			s.Equal(len(tc.expected.cveNames), len(reportData.cvss))

			reportSnap = testReportSnapshot(tc.collection.GetId(), tc.fixability, tc.severities, tc.imageTypes, tc.scopeRules)
			startTime = time.Now()
			deployedImgResults, watchedImgResults, err := s.reportGenerator.getReportData(reportSnap, tc.collection, nil)
			graphQLDelta := time.Since(startTime).Milliseconds()
			s.NoError(err)
			reportData = extractVulnReportData(deployedImgResults, watchedImgResults)
			s.ElementsMatch(tc.expected.deploymentNames, reportData.deploymentNames)
			s.ElementsMatch(tc.expected.imageNames, reportData.imageNames)
			s.ElementsMatch(tc.expected.componentNames, reportData.componentNames)
			s.ElementsMatch(tc.expected.cveNames, reportData.cveNames)
			s.Equal(len(tc.expected.cveNames), len(reportData.cvss))

			fmt.Printf("SQF: %dms, GraphQL: %dms\n", sqfDelta, graphQLDelta)
		})
	}
}

func (s *EnhancedReportingSQFTestSuite) truncateTable(name string) {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", name)
	_, err := s.testDB.Exec(s.ctx, sql)
	s.NoError(err)
}

func (s *EnhancedReportingSQFTestSuite) upsertManyImages(images []*storage.Image) {
	for _, img := range images {
		err := s.resolver.ImageDataStore.UpsertImage(s.ctx, img)
		s.NoError(err)
	}
}

func (s *EnhancedReportingSQFTestSuite) upsertManyWatchedImages(images []*storage.Image) {
	for _, img := range images {
		err := s.watchedImageDatastore.UpsertWatchedImage(s.ctx, img.Name.FullName)
		s.NoError(err)
	}
}

func (s *EnhancedReportingSQFTestSuite) upsertManyDeployments(deployments []*storage.Deployment) {
	for _, dep := range deployments {
		err := s.resolver.DeploymentDataStore.UpsertDeployment(s.ctx, dep)
		s.NoError(err)
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

func extractVulnReportDataSQF(cveResponses []*ImageCVEQueryResponse) *vulnReportData {
	deploymentNames := set.NewStringSet()
	imageNames := set.NewStringSet()
	componentNames := set.NewStringSet()
	cveNames := make([]string, 0)
	cvss := make([]float64, 0)

	for _, res := range cveResponses {
		if res.Deployment != "" {
			deploymentNames.Add(res.Deployment)
		}
		imageNames.Add(res.Image)
		componentNames.Add(res.Component)
		cveNames = append(cveNames, res.CVE)
		cvss = append(cvss, res.CVSS)
	}
	return &vulnReportData{
		deploymentNames: deploymentNames.AsSlice(),
		imageNames:      imageNames.AsSlice(),
		componentNames:  componentNames.AsSlice(),
		cveNames:        cveNames,
		cvss:            cvss,
	}
}

func extractVulnReportData(deployedImgResults []common.DeployedImagesResult, watchedImgResults []common.WatchedImagesResult) *vulnReportData {
	deploymentNames := make([]string, 0)
	imageNames := make([]string, 0)
	componentNames := make([]string, 0)
	cveNames := make([]string, 0)
	cvss := make([]float64, 0)

	for _, res := range deployedImgResults {
		for _, dep := range res.Deployments {
			deploymentNames = append(deploymentNames, dep.DeploymentName)
			for _, img := range dep.Images {
				imageNames = append(imageNames, img.Name.FullName)
				for _, comp := range img.ImageComponents {
					componentNames = append(componentNames, comp.Name)
					for _, cve := range comp.ImageVulnerabilities {
						cveNames = append(cveNames, cve.Cve)
						cvss = append(cvss, cve.Cvss)
					}
				}
			}
		}
	}
	for _, res := range watchedImgResults {
		for _, img := range res.Images {
			imageNames = append(imageNames, img.Name.FullName)
			for _, comp := range img.ImageComponents {
				componentNames = append(componentNames, comp.Name)
				for _, cve := range comp.ImageVulnerabilities {
					cveNames = append(cveNames, cve.Cve)
					cvss = append(cvss, cve.Cvss)
				}
			}
		}
	}

	return &vulnReportData{
		deploymentNames: deploymentNames,
		imageNames:      imageNames,
		componentNames:  componentNames,
		cveNames:        cveNames,
		cvss:            cvss,
	}
}
