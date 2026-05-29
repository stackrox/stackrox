//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testutils"
	pkgTestUtils "github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestImageIntegrationDatastoreSAC(t *testing.T) {
	suite.Run(t, new(imageIntegrationDatastoreSACSuite))
}

type imageIntegrationDatastoreSACSuite struct {
	suite.Suite

	datastore  DataStore
	pgTestBase *pgtest.TestPostgres

	testContexts      map[string]context.Context
	testIntegrationID string
}

func (s *imageIntegrationDatastoreSACSuite) SetupSuite() {
	s.pgTestBase = pgtest.ForT(s.T())
	s.Require().NotNil(s.pgTestBase)

	s.datastore = GetTestPostgresDataStore(s.T(), s.pgTestBase.DB)

	// Setup test contexts with Integration as the controlled resource and Administration as an alternative
	s.testContexts = testutils.GetGloballyScopedTestContexts(context.Background(), s.T(), resources.Administration, resources.Integration)
}

func (s *imageIntegrationDatastoreSACSuite) TearDownSuite() {
	s.pgTestBase.Close()
}

func (s *imageIntegrationDatastoreSACSuite) SetupTest() {
	s.testIntegrationID = ""

	// Clean up all image integrations before each test to ensure clean state
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	tag, err := s.pgTestBase.Exec(unrestrictedCtx, "TRUNCATE image_integrations CASCADE")
	s.T().Log("image_integrations", tag)
	s.NoError(err)
}

func (s *imageIntegrationDatastoreSACSuite) TearDownTest() {
	// Clean up all test image integrations after each test
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	tag, err := s.pgTestBase.Exec(unrestrictedCtx, "TRUNCATE image_integrations CASCADE")
	s.T().Log("image_integrations", tag)
	s.NoError(err)
}

// Helper functions for creating test objects

func imageIntegrationFactory() *storage.ImageIntegration {
	integration := &storage.ImageIntegration{}
	err := pkgTestUtils.FullInit(integration, pkgTestUtils.SimpleInitializer(), pkgTestUtils.JSONFieldsFilter)
	if err != nil {
		panic(err)
	}
	// Set a specific integration config to make it valid
	integration.IntegrationConfig = &storage.ImageIntegration_Docker{
		Docker: &storage.DockerConfig{
			Endpoint: "https://docker.io",
		},
	}
	return integration
}

func uniqueImageIntegrationFactory() *storage.ImageIntegration {
	integration := &storage.ImageIntegration{}
	err := pkgTestUtils.FullInit(integration, pkgTestUtils.UniqueInitializer(), pkgTestUtils.JSONFieldsFilter)
	if err != nil {
		panic(err)
	}
	// Set a specific integration config to make it valid
	integration.IntegrationConfig = &storage.ImageIntegration_Docker{
		Docker: &storage.DockerConfig{
			Endpoint: "https://docker.io",
		},
	}
	return integration
}

func mutateImageIntegration(integration *storage.ImageIntegration) *storage.ImageIntegration {
	mutated := integration.CloneVT()
	mutated.Name = "updated-" + integration.GetName()
	return mutated
}

func imageIntegrationIDExtractor(integration *storage.ImageIntegration) string {
	return integration.GetId()
}

func (s *imageIntegrationDatastoreSACSuite) addImageIntegration(ctx context.Context, integration *storage.ImageIntegration) error {
	id, err := s.datastore.AddImageIntegration(ctx, integration)
	if err != nil {
		return err
	}
	// Update the integration ID with the returned ID
	integration.Id = id
	return nil
}

func (s *imageIntegrationDatastoreSACSuite) updateImageIntegration(ctx context.Context, integration *storage.ImageIntegration) error {
	return s.datastore.UpdateImageIntegration(ctx, integration)
}

// Tests for read operations
// Note: Read operations use globally scoped store which filters results based on SAC
// rather than returning access denied errors

func (s *imageIntegrationDatastoreSACSuite) TestGetImageIntegration() {
	testCases := testutils.GenericGlobalSACReadTestCasesNoAccessNoError("get image integration")

	testutils.RunGetTests(
		s.T(),
		testCases,
		s.testContexts,
		imageIntegrationIDExtractor,
		imageIntegrationFactory,
		s.addImageIntegration,
		s.datastore.GetImageIntegration,
		s.datastore.RemoveImageIntegration,
	)
}

func (s *imageIntegrationDatastoreSACSuite) TestGetImageIntegrations() {
	testCases := testutils.GenericGlobalSACReadTestCasesNoAccessNoError("get all image integrations")

	testutils.RunGetAllTests(
		s.T(),
		testCases,
		s.testContexts,
		imageIntegrationIDExtractor,
		uniqueImageIntegrationFactory,
		s.addImageIntegration,
		func(ctx context.Context) ([]*storage.ImageIntegration, error) {
			return s.datastore.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{})
		},
		s.datastore.RemoveImageIntegration,
	)
}

// Tests for write operations

func (s *imageIntegrationDatastoreSACSuite) TestAddImageIntegration() {
	testCases := testutils.GenericGlobalSACWriteTestCases("add image integration")

	testutils.RunAddTests(
		s.T(),
		testCases,
		s.testContexts,
		imageIntegrationIDExtractor,
		uniqueImageIntegrationFactory,
		s.addImageIntegration,
		s.datastore.GetImageIntegration,
		s.datastore.RemoveImageIntegration,
	)
}

func (s *imageIntegrationDatastoreSACSuite) TestUpdateImageIntegration() {
	testCases := testutils.GenericGlobalSACWriteTestCases("update image integration")

	testutils.RunUpdateTests(
		s.T(),
		testCases,
		s.testContexts,
		imageIntegrationIDExtractor,
		imageIntegrationFactory,
		s.addImageIntegration,
		mutateImageIntegration,
		s.updateImageIntegration,
		s.datastore.GetImageIntegration,
		s.datastore.RemoveImageIntegration,
	)
}

func (s *imageIntegrationDatastoreSACSuite) TestRemoveImageIntegration() {
	testCases := testutils.GenericGlobalSACWriteTestCases("remove image integration")

	testutils.RunRemoveTests(
		s.T(),
		testCases,
		s.testContexts,
		imageIntegrationIDExtractor,
		imageIntegrationFactory,
		s.addImageIntegration,
		s.datastore.GetImageIntegration,
		s.datastore.RemoveImageIntegration,
	)
}

// Tests for search operations
// The tested operations delegate to the store which has SAC built-in

func (s *imageIntegrationDatastoreSACSuite) TestSearch() {
	testCases := testutils.GenericGlobalSACReadTestCasesNoAccessNoError("search")

	testutils.RunSearchTests(
		s.T(),
		testCases,
		s.testContexts,
		imageIntegrationIDExtractor,
		uniqueImageIntegrationFactory,
		s.addImageIntegration,
		s.datastore.Search,
		s.datastore.RemoveImageIntegration,
	)
}

func (s *imageIntegrationDatastoreSACSuite) TestSearchImageIntegrations() {
	testCases := testutils.GenericGlobalSACReadTestCasesNoAccessNoError("search image integrations")

	testutils.RunSearchResultsTests(
		s.T(),
		testCases,
		s.testContexts,
		imageIntegrationIDExtractor,
		uniqueImageIntegrationFactory,
		s.addImageIntegration,
		s.datastore.SearchImageIntegrations,
		s.datastore.RemoveImageIntegration,
	)
}

func (s *imageIntegrationDatastoreSACSuite) TestCount() {
	testCases := testutils.GenericGlobalSACReadTestCasesNoAccessNoError("count")

	testutils.RunCountTests(
		s.T(),
		testCases,
		s.testContexts,
		imageIntegrationIDExtractor,
		uniqueImageIntegrationFactory,
		s.addImageIntegration,
		s.datastore.Count,
		s.datastore.RemoveImageIntegration,
	)
}
