//go:build sql_integration

package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/reports/common"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	collectionSearch "github.com/stackrox/rox/central/resourcecollection/datastore/search"
	collectionPostgres "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	types2 "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestReportingWithCollections(t *testing.T) {
	suite.Run(t, new(ReportingWithCollectionsTestSuite))
}

type ReportingWithCollectionsTestSuite struct {
	suite.Suite

	ctx             context.Context
	testDB          *pgtest.TestPostgres
	reportScheduler *scheduler
	resolver        *resolvers.Resolver
	schema          *graphql.Schema

	collectionDatastore     collectionDS.DataStore
	collectionQueryResolver collectionDS.QueryResolver
}

type vulnReportData struct {
	deploymentNames []string
	imageNames      []string
	componentNames  []string
	cveNames        []string
}

func (s *ReportingWithCollectionsTestSuite) SetupSuite() {

	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = resolvers.SetupTestPostgresConn(s.T())
	imageDataStore := resolvers.CreateTestImageDatastore(s.T(), s.testDB, mockCtrl)
	s.resolver, s.schema = resolvers.SetupTestResolver(s.T(),
		imageDataStore,
		resolvers.CreateTestImageComponentDatastore(s.T(), s.testDB, mockCtrl),
		resolvers.CreateTestImageCVEDatastore(s.T(), s.testDB),
		resolvers.CreateTestImageComponentCVEEdgeDatastore(s.T(), s.testDB),
		resolvers.CreateTestImageCVEEdgeDatastore(s.T(), s.testDB),
		resolvers.CreateTestDeploymentDatastore(s.T(), s.testDB, mockCtrl, imageDataStore),
	)

	var err error
	collectionStore := collectionPostgres.CreateTableAndNewStore(s.ctx, s.testDB.DB, s.testDB.GetGormDB(s.T()))
	index := collectionPostgres.NewIndexer(s.testDB.DB)
	s.collectionDatastore, s.collectionQueryResolver, err = collectionDS.New(collectionStore, collectionSearch.New(collectionStore, index))
	s.NoError(err)

	s.reportScheduler = newSchedulerImpl(nil, nil, nil, nil,
		s.resolver.DeploymentDataStore, s.collectionDatastore, nil, s.collectionQueryResolver,
		nil, nil, s.schema)
}

func (s *ReportingWithCollectionsTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *ReportingWithCollectionsTestSuite) TearDownTest() {
	s.truncateTable(postgresSchema.DeploymentsTableName)
	s.truncateTable(postgresSchema.ImagesTableName)
	s.truncateTable(postgresSchema.ImageComponentsTableName)
	s.truncateTable(postgresSchema.ImageCvesTableName)
	s.truncateTable(postgresSchema.CollectionsTableName)
}

func (s *ReportingWithCollectionsTestSuite) TestGetReportData() {
	ctx := resolvers.SetAuthorizerOverride(s.ctx, allow.Anonymous())
	clusters := []string{"c1", "c2"}
	namespaces := []string{"ns1", "ns2"}
	deployments, images := testDeploymentsWithImages(clusters, namespaces, 1)
	s.upsertManyImages(images)
	s.upsertManyDeployments(deployments)

	testCases := []struct {
		name       string
		collection *storage.ResourceCollection
		fixability storage.VulnerabilityReportFilters_Fixability
		severities []storage.VulnerabilitySeverity
		expected   *vulnReportData
	}{
		{
			name:       "Include all deployments; CVEs with both fixabilities and all severities",
			collection: testCollection("col1", "", "", ""),
			fixability: storage.VulnerabilityReportFilters_BOTH,
			severities: []storage.VulnerabilitySeverity{
				storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
				storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			},
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
			name:       "Include all deployments; Fixable CVEs with CRITICAL severity",
			collection: testCollection("col2", "", "", ""),
			fixability: storage.VulnerabilityReportFilters_FIXABLE,
			severities: []storage.VulnerabilitySeverity{
				storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			},
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
			name:       "Include deployments from cluster c1 and namespace ns1; CVEs with both fixabilities and all severities",
			collection: testCollection("col3", "c1", "ns1", ""),
			fixability: storage.VulnerabilityReportFilters_BOTH,
			severities: []storage.VulnerabilitySeverity{
				storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
				storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
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
	}

	for _, c := range testCases {
		_, err := s.collectionDatastore.AddCollection(s.ctx, c.collection)
		s.NoError(err)

		reportConfig := testReportConfig(c.collection.GetId(), c.fixability, c.severities)
		results, err := s.reportScheduler.getReportData(ctx, reportConfig)
		s.NoError(err)
		reportData := extractVulnReportData(results)
		s.ElementsMatch(c.expected.deploymentNames, reportData.deploymentNames)
		s.ElementsMatch(c.expected.imageNames, reportData.imageNames)
		s.ElementsMatch(c.expected.componentNames, reportData.componentNames)
		s.ElementsMatch(c.expected.cveNames, reportData.cveNames)
	}
}

func (s *ReportingWithCollectionsTestSuite) truncateTable(name string) {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", name)
	_, err := s.testDB.Exec(s.ctx, sql)
	s.NoError(err)
}

func (s *ReportingWithCollectionsTestSuite) upsertManyImages(images []*storage.Image) {
	for _, img := range images {
		err := s.resolver.ImageDataStore.UpsertImage(s.ctx, img)
		s.NoError(err)
	}
}

func (s *ReportingWithCollectionsTestSuite) upsertManyDeployments(deployments []*storage.Deployment) {
	for _, dep := range deployments {
		err := s.resolver.DeploymentDataStore.UpsertDeployment(s.ctx, dep)
		s.NoError(err)
	}
}

func testDeploymentsWithImages(clusters, namespaces []string, numDeploymentsPerNamespace int) ([]*storage.Deployment, []*storage.Image) {
	capacity := len(clusters) * len(namespaces) * numDeploymentsPerNamespace
	deployments := make([]*storage.Deployment, 0, capacity)
	images := make([]*storage.Image, 0, capacity)
	for _, cluster := range clusters {
		for _, namespace := range namespaces {
			for i := 0; i < numDeploymentsPerNamespace; i++ {
				depName := fmt.Sprintf("%s_%s_dep%d", cluster, namespace, i)
				image := testImage(depName)
				deployment := testDeployment(depName, cluster, namespace, image)
				deployments = append(deployments, deployment)
				images = append(images, image)
			}
		}
	}
	return deployments, images
}

func testDeployment(deploymentName, cluster, namespace string, image *storage.Image) *storage.Deployment {
	return &storage.Deployment{
		Name:        deploymentName,
		Id:          uuid.NewV4().String(),
		ClusterName: cluster,
		ClusterId:   uuid.NewV4().String(),
		Namespace:   namespace,
		NamespaceId: uuid.NewV4().String(),
		Containers: []*storage.Container{
			{
				Name:  fmt.Sprintf("%s_container", deploymentName),
				Image: types2.ToContainerImage(image),
			},
		},
	}
}

func testImage(deployment string) *storage.Image {
	t, err := ptypes.TimestampProto(time.Unix(0, 1000))
	utils.CrashOnError(err)
	return &storage.Image{
		Id:   fmt.Sprintf("%s_img", deployment),
		Name: &storage.ImageName{FullName: fmt.Sprintf("%s_img", deployment)},
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
					Name:    fmt.Sprintf("%s_img_comp", deployment),
					Version: "1.0",
					Vulns: []*storage.EmbeddedVulnerability{
						{
							Cve: fmt.Sprintf("CVE-fixable_critical-%s_img_comp", deployment),
							SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
								FixedBy: "1.1",
							},
							Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
							Link:     "link",
						},
						{
							Cve:      fmt.Sprintf("CVE-nonFixable_low-%s_img_comp", deployment),
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

func testReportConfig(collectionID string, fixability storage.VulnerabilityReportFilters_Fixability,
	severities []storage.VulnerabilitySeverity) *storage.ReportConfiguration {
	config := fixtures.GetValidReportConfiguration()
	config.Filter = &storage.ReportConfiguration_VulnReportFilters{
		VulnReportFilters: &storage.VulnerabilityReportFilters{
			Fixability:      fixability,
			SinceLastReport: false,
			Severities:      severities,
		},
	}
	config.ScopeId = collectionID
	return config
}

func extractVulnReportData(results []common.DeployedImagesResult) *vulnReportData {
	deploymentNames := make([]string, 0)
	imageNames := make([]string, 0)
	componentNames := make([]string, 0)
	cveNames := make([]string, 0)

	for _, res := range results {
		for _, dep := range res.Deployments {
			deploymentNames = append(deploymentNames, dep.DeploymentName)
			for _, img := range dep.Images {
				imageNames = append(imageNames, img.Name.FullName)
				for _, comp := range img.ImageComponents {
					componentNames = append(componentNames, comp.Name)
					for _, cve := range comp.ImageVulnerabilities {
						cveNames = append(cveNames, cve.Cve)
					}
				}
			}
		}
	}

	return &vulnReportData{
		deploymentNames: deploymentNames,
		imageNames:      imageNames,
		componentNames:  componentNames,
		cveNames:        cveNames,
	}
}
