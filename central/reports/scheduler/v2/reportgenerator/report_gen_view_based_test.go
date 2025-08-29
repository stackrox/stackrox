//go:build sql_integration

package reportgenerator

import (
	"context"
	"fmt"
	"testing"

	blobDS "github.com/stackrox/rox/central/blob/datastore"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	namespaceDSMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	imagesView "github.com/stackrox/rox/central/views/images"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestViewBasedReporting(t *testing.T) {
	suite.Run(t, new(ViewBasedReportingTestSuite))
}

type ViewBasedReportingTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller

	ctx                   context.Context
	testDB                *pgtest.TestPostgres
	reportGenerator       *reportGeneratorImpl
	resolver              *resolvers.Resolver
	watchedImageDatastore watchedImageDS.DataStore
	clusterDatastore      *clusterDSMocks.MockDataStore
	namespaceDatastore    *namespaceDSMocks.MockDataStore
	blobStore             blobDS.Datastore
}

type viewBasedReportData struct {
	deploymentNames []string
	imageNames      []string
	componentNames  []string
	cveNames        []string
	cvss            []float64
}

func (s *ViewBasedReportingTestSuite) SetupSuite() {
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	s.testDB = resolvers.SetupTestPostgresConn(s.T())
}

func (s *ViewBasedReportingTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	// Create data stores based on feature flag
	imageDataStore := resolvers.CreateTestImageV2Datastore(s.T(), s.testDB, s.mockCtrl)
	s.resolver, _ = resolvers.SetupTestResolver(s.T(),
		imagesView.NewImageView(s.testDB.DB),
		imageDataStore,
		resolvers.CreateTestImageComponentV2Datastore(s.T(), s.testDB, s.mockCtrl),
		resolvers.CreateTestImageCVEV2Datastore(s.T(), s.testDB),
		resolvers.CreateTestDeploymentDatastore(s.T(), s.testDB, s.mockCtrl, imageDataStore),
		deploymentsView.NewDeploymentView(s.testDB.DB),
	)

	s.watchedImageDatastore = watchedImageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.clusterDatastore = clusterDSMocks.NewMockDataStore(s.mockCtrl)
	s.namespaceDatastore = namespaceDSMocks.NewMockDataStore(s.mockCtrl)
	s.blobStore = blobDS.NewTestDatastore(s.T(), s.testDB.DB)

	s.reportGenerator = newReportGeneratorImpl(s.testDB, nil, s.resolver.DeploymentDataStore,
		s.watchedImageDatastore, nil, nil, s.blobStore, s.clusterDatastore,
		s.namespaceDatastore, s.resolver.ImageCVEDataStore, s.resolver.ImageCVEV2DataStore, nil)
}

func (s *ViewBasedReportingTestSuite) TearDownTest() {
	s.mockCtrl.Finish()

	s.truncateTable(postgresSchema.DeploymentsTableName)
	s.truncateTable(postgresSchema.ImagesTableName)
	s.truncateTable(postgresSchema.ImageComponentV2TableName)
	s.truncateTable(postgresSchema.ImageCvesV2TableName)

	s.truncateTable(postgresSchema.CollectionsTableName)
}

func (s *ViewBasedReportingTestSuite) TestGetReportDataViewBased() {
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
		name     string
		query    string
		expected *viewBasedReportData
	}{
		{
			name:  "View-based report with CVE severity filter",
			query: "SEVERITY:CRITICAL_VULNERABILITY_SEVERITY",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp",
					"CVE-fixable_critical-c2_ns1_dep0_img_comp",
					"CVE-fixable_critical-c2_ns2_dep0_img_comp",
					"CVE-fixable_critical-w0_img_comp",
					"CVE-fixable_critical-w1_img_comp",
				},
			},
		},
		{
			name:  "View-based report with critical important severity filter",
			query: "SEVERITY:CRITICAL_VULNERABILITY_SEVERITY,IMPORTANT_VULNERABILITY_SEVERITY",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames: []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp",
					"w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp",
					"CVE-fixable_critical-c2_ns1_dep0_img_comp",
					"CVE-fixable_critical-c2_ns2_dep0_img_comp",
					"CVE-fixable_critical-w0_img_comp",
					"CVE-fixable_critical-w1_img_comp",
				},
			},
		},
		{
			name:  "View-based report with complex query combining multiple fields",
			query: "CVE Severity:CRITICAL_VULNERABILITY_SEVERITY+Fixable:true",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp",
					"CVE-fixable_critical-c2_ns1_dep0_img_comp",
					"CVE-fixable_critical-c2_ns2_dep0_img_comp",
					"CVE-fixable_critical-w0_img_comp",
					"CVE-fixable_critical-w1_img_comp",
				},
			},
		},
		{
			name:  "View-based report with empty query (should return all vulnerabilities)",
			query: "",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp", "CVE-nonFixable_low-c1_ns2_dep0_img_comp",
					"CVE-fixable_critical-c2_ns1_dep0_img_comp", "CVE-nonFixable_low-c2_ns1_dep0_img_comp",
					"CVE-fixable_critical-c2_ns2_dep0_img_comp", "CVE-nonFixable_low-c2_ns2_dep0_img_comp",
					"CVE-fixable_critical-w0_img_comp",
					"CVE-nonFixable_low-w0_img_comp",
					"CVE-fixable_critical-w1_img_comp",
					"CVE-nonFixable_low-w1_img_comp",
				},
			},
		},
		// Test cases for CVE-specific search fields
		{
			name:  "View-based report filtering by CVE ID",
			query: "CVE:CVE-fixable_critical-c1_ns1_dep0_img_comp",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp"},
				cveNames:        []string{"CVE-fixable_critical-c1_ns1_dep0_img_comp"},
			},
		},
		{
			name:  "View-based report filtering by CVSS score range",
			query: "CVSS:>=7.0",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp",
					"CVE-fixable_critical-c2_ns1_dep0_img_comp",
					"CVE-fixable_critical-c2_ns2_dep0_img_comp",
					"CVE-fixable_critical-w0_img_comp",
					"CVE-fixable_critical-w1_img_comp",
				},
			},
		},
		{
			name:  "View-based report filtering by NVD CVSS score",
			query: "NVD CVSS:>=8",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp",
					"CVE-fixable_critical-c2_ns1_dep0_img_comp",
					"CVE-fixable_critical-c2_ns2_dep0_img_comp",
					"CVE-fixable_critical-w0_img_comp",
					"CVE-fixable_critical-w1_img_comp",
				},
			},
		},
		{
			name:  "View-based report filtering by vulnerability state",
			query: "Vulnerability State:OBSERVED",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
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
		// Test cases for component-related search fields
		{
			name:  "View-based report filtering by component name",
			query: "Component:c1_ns1_dep0_img_comp",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp"},
				cveNames:        []string{"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp"},
			},
		},
		// Test cases for image-related search fields
		{
			name:  "View-based report filtering by image name",
			query: "Image:c1_ns1_dep0_img",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp"},
				cveNames:        []string{"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp"},
			},
		},
		{
			name:  "View-based report filtering by image registry",
			query: "Image Registry:docker.io",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
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
			name:  "View-based report filtering by image tag",
			query: "Image Tag:latest",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
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
		// Test cases for deployment-related search fields
		{
			name:  "View-based report filtering by cluster name",
			query: "Cluster:r/c1",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp", "CVE-nonFixable_low-c1_ns2_dep0_img_comp",
				},
			},
		},
		{
			name:  "View-based report filtering by namespace",
			query: "Namespace:r/ns1",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c2_ns1_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c2_ns1_dep0_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c2_ns1_dep0_img_comp"},
				cveNames:        []string{"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp", "CVE-fixable_critical-c2_ns1_dep0_img_comp", "CVE-nonFixable_low-c2_ns1_dep0_img_comp"},
			},
		},
		{
			name:  "View-based report filtering by deployment name",
			query: "Deployment:r/c1_ns1_dep0",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp"},
				cveNames:        []string{"CVE-fixable_critical-c1_ns1_dep0_img_comp", "CVE-nonFixable_low-c1_ns1_dep0_img_comp"},
			},
		},
		// Test cases for fixability-related search fields
		{
			name:  "View-based report filtering by non-fixable vulnerabilities",
			query: "Fixable:false",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-nonFixable_low-c1_ns1_dep0_img_comp",
					"CVE-nonFixable_low-c1_ns2_dep0_img_comp",
					"CVE-nonFixable_low-c2_ns1_dep0_img_comp",
					"CVE-nonFixable_low-c2_ns2_dep0_img_comp",
					"CVE-nonFixable_low-w0_img_comp",
					"CVE-nonFixable_low-w1_img_comp",
				},
			},
		},
		// Test cases for combinations of multiple search fields
		{
			name:  "View-based report with multiple field combination",
			query: "Cluster:c1+Severity:CRITICAL_VULNERABILITY_SEVERITY+Fixable:true",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp",
				},
			},
		},
		// Test cases for advanced search fields - EPSS and Advisory fields
		{
			name:  "View-based report filtering by EPSS Probability",
			query: "EPSS Probability:>=0.0",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
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
			name:  "View-based report filtering by Fixed By version",
			query: "Fixed By:1.1",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp",
					"CVE-fixable_critical-c2_ns1_dep0_img_comp",
					"CVE-fixable_critical-c2_ns2_dep0_img_comp",
					"CVE-fixable_critical-w0_img_comp",
					"CVE-fixable_critical-w1_img_comp",
				},
			},
		},
		// Test cases for component version field
		{
			name:  "View-based report filtering by component version",
			query: "Component Version:1.0",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
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
		// Test cases for timestamp-based search fields
		{
			name:  "View-based report filtering by first image occurrence timestamp range",
			query: "First Image Occurrence Timestamp:<01/01/2020",
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0", "c2_ns2_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "c2_ns2_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "c2_ns2_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
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
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			log.Infof("Running test %s", tc.name)
			reportSnap := testViewBasedReportSnapshot(tc.query, nil)
			// Test get data using view-based approach
			reportData, err := s.reportGenerator.getReportDataViewBased(reportSnap)
			s.NoError(err)
			collected := s.collectViewBasedReportData(reportData.CVEResponses)
			s.ElementsMatch(tc.expected.deploymentNames, collected.deploymentNames)
			s.ElementsMatch(tc.expected.imageNames, collected.imageNames)
			s.ElementsMatch(tc.expected.componentNames, collected.componentNames)
			s.ElementsMatch(tc.expected.cveNames, collected.cveNames)
			s.Equal(len(tc.expected.cveNames), reportData.NumDeployedImageResults+reportData.NumWatchedImageResults)
			s.Equal(len(tc.expected.cveNames), len(collected.cvss))
		})
	}
}

func (s *ViewBasedReportingTestSuite) TestGetReportDataViewBasedWithInvalidQuery() {
	clusters := []*storage.Cluster{
		{Id: uuid.NewV4().String(), Name: "c1"},
	}

	namespaces := testNamespaces(clusters, 1)
	deployments, images := testDeploymentsWithImages(namespaces, 1)
	s.upsertManyImages(images)
	s.upsertManyDeployments(deployments)

	s.clusterDatastore.EXPECT().GetClusters(gomock.Any()).
		Return(clusters, nil).AnyTimes()
	s.namespaceDatastore.EXPECT().GetAllNamespaces(gomock.Any()).
		Return(namespaces, nil).AnyTimes()

	testCases := []struct {
		name      string
		query     string
		expectErr bool
	}{
		{
			name:      "Invalid field in query",
			query:     "InvalidField:value",
			expectErr: false, // Invalid fields are typically ignored by the query parser
		},
		{
			name:      "Malformed query syntax",
			query:     "CVE Severity:",
			expectErr: false, // Empty values are typically handled gracefully
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			reportSnap := testViewBasedReportSnapshot(tc.query, nil)
			reportData, err := s.reportGenerator.getReportDataViewBased(reportSnap)
			if tc.expectErr {
				s.Error(err)
			} else {
				s.NoError(err)
				s.NotNil(reportData)
			}
		})
	}
}

func (s *ViewBasedReportingTestSuite) TestGetReportDataViewBasedAccessScope() {
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
		query      string
		scopeRules []*storage.SimpleAccessScope_Rules
		expected   *viewBasedReportData
	}{
		{
			name:  "View-based report with access scope rules",
			query: "SEVERITY:CRITICAL_VULNERABILITY_SEVERITY",
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
			expected: &viewBasedReportData{
				deploymentNames: []string{"c1_ns1_dep0", "c1_ns2_dep0", "c2_ns1_dep0"},
				imageNames:      []string{"c1_ns1_dep0_img", "c1_ns2_dep0_img", "c2_ns1_dep0_img", "w0_img", "w1_img"},
				componentNames:  []string{"c1_ns1_dep0_img_comp", "c1_ns2_dep0_img_comp", "c2_ns1_dep0_img_comp", "w0_img_comp", "w1_img_comp"},
				cveNames: []string{
					"CVE-fixable_critical-c1_ns1_dep0_img_comp",
					"CVE-fixable_critical-c1_ns2_dep0_img_comp",
					"CVE-fixable_critical-c2_ns1_dep0_img_comp",
					"CVE-fixable_critical-w0_img_comp",
					"CVE-fixable_critical-w1_img_comp",
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			reportSnap := testViewBasedReportSnapshot(tc.query, tc.scopeRules)
			// Test get data using view-based approach
			reportData, err := s.reportGenerator.getReportDataViewBased(reportSnap)
			s.NoError(err)
			collected := s.collectViewBasedReportData(reportData.CVEResponses)
			s.ElementsMatch(tc.expected.deploymentNames, collected.deploymentNames)
			s.ElementsMatch(tc.expected.imageNames, collected.imageNames)
			s.ElementsMatch(tc.expected.componentNames, collected.componentNames)
			s.ElementsMatch(tc.expected.cveNames, collected.cveNames)
			s.Equal(len(tc.expected.cveNames), reportData.NumDeployedImageResults+reportData.NumWatchedImageResults)
			s.Equal(len(tc.expected.cveNames), len(collected.cvss))
		})
	}

}

// Helper functions
func (s *ViewBasedReportingTestSuite) truncateTable(name string) {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", name)
	_, err := s.testDB.Exec(s.ctx, sql)
	s.NoError(err)
}

func (s *ViewBasedReportingTestSuite) upsertManyImages(images []*storage.Image) {
	for _, img := range images {
		err := s.resolver.ImageDataStore.UpsertImage(s.ctx, img)
		s.NoError(err)
	}
}

func (s *ViewBasedReportingTestSuite) upsertManyWatchedImages(images []*storage.Image) {
	for _, img := range images {
		err := s.watchedImageDatastore.UpsertWatchedImage(s.ctx, img.Name.FullName)
		s.NoError(err)
	}
}

func (s *ViewBasedReportingTestSuite) upsertManyDeployments(deployments []*storage.Deployment) {
	for _, dep := range deployments {
		err := s.resolver.DeploymentDataStore.UpsertDeployment(s.ctx, dep)
		s.NoError(err)
	}
}

func (s *ViewBasedReportingTestSuite) collectViewBasedReportData(cveResponses []*ImageCVEQueryResponse) *viewBasedReportData {
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
	return &viewBasedReportData{
		deploymentNames: deploymentNames.AsSlice(),
		imageNames:      imageNames.AsSlice(),
		componentNames:  componentNames.AsSlice(),
		cveNames:        cveNames,
		cvss:            cvss,
	}
}
