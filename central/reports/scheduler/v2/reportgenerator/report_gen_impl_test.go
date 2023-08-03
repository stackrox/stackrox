//go:build sql_integration

package reportgenerator

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/reports/common"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	collectionSearch "github.com/stackrox/rox/central/resourcecollection/datastore/search"
	collectionPostgres "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
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

func TestEnhancedReporting(t *testing.T) {
	suite.Run(t, new(EnhancedReportingTestSuite))
}

type EnhancedReportingTestSuite struct {
	suite.Suite

	ctx             context.Context
	testDB          *pgtest.TestPostgres
	reportGenerator *reportGeneratorImpl
	resolver        *resolvers.Resolver
	schema          *graphql.Schema

	collectionDatastore     collectionDS.DataStore
	watchedImageDatastore   watchedImageDS.DataStore
	collectionQueryResolver collectionDS.QueryResolver

	blobStore blobDS.Datastore
}

type vulnReportData struct {
	deploymentNames []string
	imageNames      []string
	componentNames  []string
	cveNames        []string
}

func (s *EnhancedReportingTestSuite) SetupSuite() {
	s.T().Setenv(features.VulnMgmtReportingEnhancements.EnvVar(), "true")
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		s.T().Skip("Skip tests when ROX_VULN_MGMT_REPORTING_ENHANCEMENTS disabled")
		s.T().SkipNow()
	}
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

	s.watchedImageDatastore = watchedImageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)

	s.blobStore = blobDS.NewTestDatastore(s.T(), s.testDB.DB)
	s.reportGenerator = newReportGeneratorImpl(nil, nil, nil,
		s.resolver.DeploymentDataStore, s.watchedImageDatastore, s.collectionDatastore, s.collectionQueryResolver,
		nil, nil, s.blobStore, s.schema)
}

func (s *EnhancedReportingTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
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
	s.Require().NoError(s.reportGenerator.saveReportData(configID, reportID, nil))
	newBuf, _, exists, err = s.blobStore.GetBlobWithDataInBuffer(s.ctx, common.GetReportBlobPath(configID, reportID))
	s.Require().NoError(err)
	s.Require().True(exists)
	s.Zero(newBuf.Len())
}

func (s *EnhancedReportingTestSuite) TestGetReportData() {
	clusters := []string{"c1", "c2"}
	namespaces := []string{"ns1", "ns2"}
	deployments, images := testDeploymentsWithImages(clusters, namespaces, 1)
	s.upsertManyImages(images)
	s.upsertManyDeployments(deployments)

	watchedImages := testWatchedImages(2)
	s.upsertManyImages(watchedImages)
	s.upsertManyWatchedImages(watchedImages)

	testCases := []struct {
		name       string
		collection *storage.ResourceCollection
		fixability storage.VulnerabilityReportFilters_Fixability
		severities []storage.VulnerabilitySeverity
		imageTypes []storage.VulnerabilityReportFilters_ImageType
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
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED},
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
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED},
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
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED},
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
			name:       "Include all deployments + watched images; CVEs with both fixabilities and all severities",
			collection: testCollection("col4", "", "", ""),
			fixability: storage.VulnerabilityReportFilters_BOTH,
			severities: []storage.VulnerabilitySeverity{
				storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
				storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			},
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{
				storage.VulnerabilityReportFilters_DEPLOYED,
				storage.VulnerabilityReportFilters_WATCHED,
			},
			expected: &vulnReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0", "", ""},
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
			name:       "Include watched images only; Fixable CVEs with CRITICAL severity",
			collection: testCollection("col5", "", "", ""),
			fixability: storage.VulnerabilityReportFilters_FIXABLE,
			severities: []storage.VulnerabilitySeverity{
				storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			},
			imageTypes: []storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_WATCHED},
			expected: &vulnReportData{
				deploymentNames: []string{"", ""},
				imageNames:      []string{"w0_img", "w1_img"},
				componentNames:  []string{"w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-w0_img_comp",
					"CVE-fixable_critical-w1_img_comp",
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			err := s.collectionDatastore.AddCollection(s.ctx, tc.collection)
			s.NoError(err)

			reportConfig := testReportConfig(tc.collection.GetId(), tc.fixability, tc.severities, tc.imageTypes)
			deployedImgResults, watchedImgResults, err := s.reportGenerator.getReportData(reportConfig, tc.collection, nil)
			s.NoError(err)
			reportData := extractVulnReportData(deployedImgResults, watchedImgResults)
			s.ElementsMatch(tc.expected.deploymentNames, reportData.deploymentNames)
			s.ElementsMatch(tc.expected.imageNames, reportData.imageNames)
			s.ElementsMatch(tc.expected.componentNames, reportData.componentNames)
			s.ElementsMatch(tc.expected.cveNames, reportData.cveNames)
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

func testReportConfig(collectionID string, fixability storage.VulnerabilityReportFilters_Fixability, severities []storage.VulnerabilitySeverity,
	imageTypes []storage.VulnerabilityReportFilters_ImageType) *storage.ReportConfiguration {
	config := fixtures.GetValidReportConfigWithMultipleNotifiers()
	config.Filter = &storage.ReportConfiguration_VulnReportFilters{
		VulnReportFilters: &storage.VulnerabilityReportFilters{
			Fixability: fixability,
			Severities: severities,
			ImageTypes: imageTypes,
			CvesSince: &storage.VulnerabilityReportFilters_AllVuln{
				AllVuln: true,
			},
		},
	}
	config.ResourceScope = &storage.ResourceScope{
		ScopeReference: &storage.ResourceScope_CollectionId{
			CollectionId: collectionID,
		},
	}
	return config
}

func extractVulnReportData(deployedImgResults []common.DeployedImagesResult, watchedImgResults []common.WatchedImagesResult) *vulnReportData {
	deploymentNames := make([]string, 0)
	imageNames := make([]string, 0)
	componentNames := make([]string, 0)
	cveNames := make([]string, 0)

	for _, res := range deployedImgResults {
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
	for _, res := range watchedImgResults {
		for _, img := range res.Images {
			deploymentNames = append(deploymentNames, "")
			imageNames = append(imageNames, img.Name.FullName)
			for _, comp := range img.ImageComponents {
				componentNames = append(componentNames, comp.Name)
				for _, cve := range comp.ImageVulnerabilities {
					cveNames = append(cveNames, cve.Cve)
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
