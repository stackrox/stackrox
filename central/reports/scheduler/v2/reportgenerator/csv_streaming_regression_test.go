//go:build sql_integration

package reportgenerator

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	blobDS "github.com/stackrox/rox/central/blob/datastore"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	collectionPostgres "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	imagesView "github.com/stackrox/rox/central/views/images"
	watchedImageDS "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestStreamingCSVRegression(t *testing.T) {
	suite.Run(t, new(StreamingCSVRegressionSuite))
}

type StreamingCSVRegressionSuite struct {
	suite.Suite

	ctx                context.Context
	testDB             *pgtest.TestPostgres
	reportGenerator    *reportGeneratorImpl
	namespaceDatastore namespaceDS.DataStore
	clusterDatastore   *clusterDSMocks.MockDataStore

	clusters   []*storage.Cluster
	collection *storage.ResourceCollection
}

func (s *StreamingCSVRegressionSuite) SetupSuite() {
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = pgtest.ForT(s.T())

	watchedImageDatastore := watchedImageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)

	var resolver *resolvers.Resolver
	var schema *graphql.Schema
	if features.FlattenImageData.Enabled() {
		imgV2DataStore := resolvers.CreateTestImageV2Datastore(s.T(), s.testDB, mockCtrl)
		resolver, schema = resolvers.SetupTestResolver(s.T(),
			imagesView.NewImageView(s.testDB.DB),
			imgV2DataStore,
			resolvers.CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl),
			resolvers.CreateTestImageCVEV2Datastore(s.T(), s.testDB),
			resolvers.CreateTestDeploymentDatastoreWithImageV2(s.T(), s.testDB, mockCtrl, imgV2DataStore),
			deploymentsView.NewDeploymentView(s.testDB.DB),
		)
	} else {
		imageDataStore := resolvers.CreateTestImageDatastore(s.T(), s.testDB, mockCtrl)
		resolver, schema = resolvers.SetupTestResolver(s.T(),
			imagesView.NewImageView(s.testDB.DB),
			imageDataStore,
			resolvers.CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl),
			resolvers.CreateTestImageCVEV2Datastore(s.T(), s.testDB),
			resolvers.CreateTestDeploymentDatastore(s.T(), s.testDB, mockCtrl, imageDataStore),
			deploymentsView.NewDeploymentView(s.testDB.DB),
		)
	}

	collectionStore := collectionPostgres.New(s.testDB)
	_, collectionQueryResolver, err := collectionDS.New(collectionStore)
	s.NoError(err)
	s.clusterDatastore = clusterDSMocks.NewMockDataStore(mockCtrl)
	s.namespaceDatastore, err = namespaceDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)

	s.clusters = []*storage.Cluster{
		{Id: uuid.NewV4().String(), Name: "c1"},
		{Id: uuid.NewV4().String(), Name: "c2"},
	}

	namespaces := testNamespaces(s.clusters, 2)
	for _, ns := range namespaces {
		if ns.GetName() == "ns1" {
			ns.Labels = map[string]string{"env": "prod"}
		} else {
			ns.Labels = map[string]string{"env": "dev"}
		}
		s.Require().NoError(s.namespaceDatastore.AddNamespace(s.ctx, ns))
	}

	deployments, images := testDeploymentsWithImages(namespaces, 1)
	for _, dep := range deployments {
		s.NoError(resolver.DeploymentDataStore.UpsertDeployment(s.ctx, dep))
	}
	if features.FlattenImageData.Enabled() {
		for _, image := range images {
			s.NoError(resolver.ImageV2DataStore.UpsertImage(s.ctx, imageUtils.ConvertToV2(image)))
		}
	} else {
		for _, image := range images {
			s.NoError(resolver.ImageDataStore.UpsertImage(s.ctx, image))
		}
	}

	watchedImages := testWatchedImages(2)
	if features.FlattenImageData.Enabled() {
		for _, image := range watchedImages {
			s.NoError(resolver.ImageV2DataStore.UpsertImage(s.ctx, imageUtils.ConvertToV2(image)))
		}
	} else {
		for _, image := range watchedImages {
			s.NoError(resolver.ImageDataStore.UpsertImage(s.ctx, image))
		}
	}
	for _, img := range watchedImages {
		s.NoError(watchedImageDatastore.UpsertWatchedImage(s.ctx, img.GetName().GetFullName()))
	}

	s.clusterDatastore.EXPECT().GetClusters(gomock.Any()).
		Return(s.clusters, nil).AnyTimes()

	blobStore := blobDS.NewTestDatastore(s.T(), s.testDB.DB)

	s.reportGenerator = newReportGeneratorImpl(s.testDB, nil, resolver.DeploymentDataStore,
		watchedImageDatastore, collectionQueryResolver, nil, blobStore, s.clusterDatastore,
		s.namespaceDatastore, resolver.ImageCVEV2DataStore, schema)

	s.collection = testCollection("regression_col", "", "", "")
}

func (s *StreamingCSVRegressionSuite) TearDownSuite() {
	for _, table := range []string{
		postgresSchema.DeploymentsTableName,
		postgresSchema.ImagesTableName,
		postgresSchema.ImageComponentV2TableName,
		postgresSchema.ImageCvesV2TableName,
		postgresSchema.CollectionsTableName,
		postgresSchema.NamespacesTableName,
	} {
		_, err := s.testDB.Exec(s.ctx, fmt.Sprintf("TRUNCATE %s CASCADE", table))
		s.NoError(err)
	}
}

func (s *StreamingCSVRegressionSuite) TestCSVOutputRegression_DeployedOnly() {
	reportSnap := testReportSnapshot(s.collection.GetId(),
		storage.VulnerabilityReportFilters_BOTH,
		allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED},
		nil,
	)

	oldCSV := s.generateCSVOldPath(reportSnap, s.collection)
	newCSV := s.generateCSVStreamingPath(reportSnap, s.collection)

	s.requireCSVEqual(oldCSV, newCSV)
}

func (s *StreamingCSVRegressionSuite) TestCSVOutputRegression_DeployedAndWatched() {
	reportSnap := testReportSnapshot(s.collection.GetId(),
		storage.VulnerabilityReportFilters_BOTH,
		allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{
			storage.VulnerabilityReportFilters_DEPLOYED,
			storage.VulnerabilityReportFilters_WATCHED,
		},
		nil,
	)

	oldCSV := s.generateCSVOldPath(reportSnap, s.collection)
	newCSV := s.generateCSVStreamingPath(reportSnap, s.collection)

	s.requireCSVEqual(oldCSV, newCSV)
}

func (s *StreamingCSVRegressionSuite) TestCSVOutputRegression_FixableOnly() {
	reportSnap := testReportSnapshot(s.collection.GetId(),
		storage.VulnerabilityReportFilters_FIXABLE,
		allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{
			storage.VulnerabilityReportFilters_DEPLOYED,
			storage.VulnerabilityReportFilters_WATCHED,
		},
		nil,
	)

	oldCSV := s.generateCSVOldPath(reportSnap, s.collection)
	newCSV := s.generateCSVStreamingPath(reportSnap, s.collection)

	s.requireCSVEqual(oldCSV, newCSV)
}

func (s *StreamingCSVRegressionSuite) TestCSVOutputRegression_EmptyReport() {
	// Use a collection that matches nothing
	emptyCol := testCollection("empty_col", "nonexistent_cluster", "", "")
	reportSnap := testReportSnapshot(emptyCol.GetId(),
		storage.VulnerabilityReportFilters_BOTH,
		allSeverities(),
		[]storage.VulnerabilityReportFilters_ImageType{storage.VulnerabilityReportFilters_DEPLOYED},
		nil,
	)

	oldCSV := s.generateCSVOldPath(reportSnap, emptyCol)
	newCSV := s.generateCSVStreamingPath(reportSnap, emptyCol)

	s.requireCSVEqual(oldCSV, newCSV)
}

func (s *StreamingCSVRegressionSuite) TestCSVOutputRegression_EntityScope() {
	entityScope := &storage.EntityScope{
		Rules: []*storage.EntityScopeRule{
			{
				Entity: storage.EntityType_ENTITY_TYPE_CLUSTER,
				Field:  storage.EntityField_FIELD_NAME,
				Values: []*storage.RuleValue{
					{Value: "c1", MatchType: storage.MatchType_EXACT},
				},
			},
		},
	}
	scopeRules := []*storage.SimpleAccessScope_Rules{
		{IncludedClusters: []string{"c1"}},
	}
	allImageTypes := []storage.VulnerabilityReportFilters_ImageType{
		storage.VulnerabilityReportFilters_DEPLOYED,
		storage.VulnerabilityReportFilters_WATCHED,
	}
	reportSnap := testEntityScopeReportSnapshot(entityScope, "CVSS:>=7.0", allImageTypes, scopeRules)

	oldCSV := s.generateCSVOldPath(reportSnap, nil)
	newCSV := s.generateCSVStreamingPath(reportSnap, nil)

	s.requireCSVEqual(oldCSV, newCSV)
}

// generateCSVOldPath generates a report CSV using the old buffered path.
func (s *StreamingCSVRegressionSuite) generateCSVOldPath(snap *storage.ReportSnapshot, collection *storage.ResourceCollection) [][]string {
	reportData, err := s.reportGenerator.getReportDataSQF(snap, collection, time.Time{})
	s.Require().NoError(err)

	buf, err := GenerateCSV(reportData.CVEResponses, "regression_test")
	s.Require().NoError(err)

	rows := unzipAndParseCSV(s.T(), buf)
	return rows
}

// generateCSVStreamingPath generates a report CSV using the new streaming path.
func (s *StreamingCSVRegressionSuite) generateCSVStreamingPath(snap *storage.ReportSnapshot, collection *storage.ResourceCollection) [][]string {
	var result *StreamingReportResult
	var err error

	if snap.GetVulnReportFilters() != nil {
		result, err = s.reportGenerator.generateReportStreamingSQF(snap, collection, time.Time{}, "regression_test")
	}
	if snap.GetViewBasedVulnReportFilters() != nil {
		result, err = s.reportGenerator.generateReportStreamingViewBased(snap, "regression_test")
	}
	s.Require().NoError(err)
	s.Require().NotNil(result)

	rows := unzipAndParseCSV(s.T(), result.ZippedCSVData)
	return rows
}

// requireCSVEqual compares two parsed CSV outputs, sorting data rows to handle
// any ordering differences between old and new paths.
func (s *StreamingCSVRegressionSuite) requireCSVEqual(oldCSV, newCSV [][]string) {
	s.Require().NotEmpty(oldCSV, "old CSV should have at least a header")
	s.Require().NotEmpty(newCSV, "new CSV should have at least a header")

	// Headers must match exactly
	s.Require().Equal(oldCSV[0], newCSV[0], "CSV headers must match")

	// Sort data rows for comparison (DB order may differ between queries)
	oldData := oldCSV[1:]
	newData := newCSV[1:]
	sortCSVRows(oldData)
	sortCSVRows(newData)

	s.Require().Equal(len(oldData), len(newData), "row count must match")
	for i := range oldData {
		s.Require().Equal(oldData[i], newData[i], "row %d must match", i)
	}
}

func unzipAndParseCSV(t testing.TB, buf *bytes.Buffer) [][]string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("failed to open ZIP: %v", err)
	}
	if len(reader.File) == 0 {
		t.Fatal("ZIP contains no files")
	}
	f, err := reader.File[0].Open()
	if err != nil {
		t.Fatalf("failed to open CSV in ZIP: %v", err)
	}
	defer func() { _ = f.Close() }()

	csvData, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("failed to read CSV data: %v", err)
	}

	csvReader := csv.NewReader(bytes.NewReader(csvData))
	csvReader.FieldsPerRecord = -1
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}
	return records
}

func sortCSVRows(rows [][]string) {
	sort.Slice(rows, func(i, j int) bool {
		for col := range rows[i] {
			if col >= len(rows[j]) {
				return false
			}
			if rows[i][col] != rows[j][col] {
				return rows[i][col] < rows[j][col]
			}
		}
		return len(rows[i]) < len(rows[j])
	})
}
