//go:build sql_integration
// +build sql_integration

package datastore

import (
	"context"
	"sort"
	"testing"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/image/datastore/store/postgres"
	imageComponentDS "github.com/stackrox/rox/central/imagecomponent/datastore"
	imageComponentPostgres "github.com/stackrox/rox/central/imagecomponent/datastore/store/postgres"
	imageComponentSearch "github.com/stackrox/rox/central/imagecomponent/search"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scancomponent"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestImageDataStoreWithPostgresTestImageDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(ImagePostgresDataStoreTestSuite))
}

type ImagePostgresDataStoreTestSuite struct {
	suite.Suite

	ctx                context.Context
	db                 *pgxpool.Pool
	gormDB             *gorm.DB
	datastore          DataStore
	mockRisk           *mockRisks.MockDataStore
	componentDataStore imageComponentDS.DataStore

	envIsolator *envisolator.EnvIsolator
}

func (s *ImagePostgresDataStoreTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)

	pool, err := pgxpool.ConnectConfig(s.ctx, config)
	s.NoError(err)
	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	s.db = pool
}

func (s *ImagePostgresDataStoreTestSuite) SetupTest() {
	postgres.Destroy(s.ctx, s.db)

	s.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(s.T()))
	s.datastore = NewWithPostgres(postgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB, false), postgres.NewIndexer(s.db), s.mockRisk, ranking.ImageRanker(), ranking.ComponentRanker())

	componentStorage := imageComponentPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	componentIndexer := imageComponentPostgres.NewIndexer(s.db)
	componentSearcher := imageComponentSearch.NewV2(componentStorage, componentIndexer)
	s.componentDataStore = imageComponentDS.New(nil, componentStorage, componentIndexer, componentSearcher, s.mockRisk, ranking.ComponentRanker())
}

func (s *ImagePostgresDataStoreTestSuite) TearDownSuite() {
	s.envIsolator.RestoreAll()
	s.db.Close()
	pgtest.CloseGormDB(s.T(), s.gormDB)
}

func (s *ImagePostgresDataStoreTestSuite) TestSearchWithPostgres() {
	image := getTestImage("id1")

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Image),
	))

	// Upsert image.
	s.NoError(s.datastore.UpsertImage(ctx, image))

	// Basic unscoped search.
	results, err := s.datastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageSHA, image.GetId()).ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)

	// Upsert new image.
	newImage := getTestImage("id2")
	newImage.GetScan().Components = append(newImage.GetScan().GetComponents(), &storage.EmbeddedImageScanComponent{
		Name:    "comp3",
		Version: "ver1",
		Vulns: []*storage.EmbeddedVulnerability{
			{
				Cve:               "cve3",
				VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			},
		},
	})
	s.NoError(s.datastore.UpsertImage(ctx, newImage))

	// Search multiple images.
	images, err := s.datastore.SearchRawImages(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(images, 2)

	// Search for just one image.
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageSHA, image.GetId()).ProtoQuery()
	images, err = s.datastore.SearchRawImages(ctx, q)
	s.NoError(err)
	s.Len(images, 1)

	// Search by CVE.
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, "cve1").ProtoQuery()
	images, err = s.datastore.SearchRawImages(ctx, q)
	s.NoError(err)
	s.Len(images, 2)

	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, "cve3").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal("id2", results[0].ID)

	// Scope search by image.
	scopedCtx := scoped.Context(ctx, scoped.Scope{
		ID:    image.GetId(),
		Level: v1.SearchCategory_IMAGES,
	})
	results, err = s.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(image.GetId(), results[0].ID)

	// Scope search by vulns.
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "cve3#blah",
		Level: v1.SearchCategory_IMAGE_VULNERABILITIES,
	})
	results, err = s.datastore.Search(scopedCtx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(newImage.GetId(), results[0].ID)
}

func (s *ImagePostgresDataStoreTestSuite) TestFixableWithPostgres() {
	image := fixtures.GetImageWithUniqueComponents()
	ctx := sac.WithAllAccess(context.Background())

	s.NoError(s.datastore.UpsertImage(ctx, image))
	_, found, err := s.datastore.GetImage(ctx, image.GetId())
	s.NoError(err)
	s.True(found)

	results, err := s.datastore.Search(ctx, pkgSearch.NewQueryBuilder().AddBools(pkgSearch.Fixable, true).ProtoQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(image.GetId(), results[0].ID)

	image.Scan.ScanTime = protoTypes.TimestampNow()
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			vuln.SetFixedBy = nil
		}
	}
	s.NoError(s.datastore.UpsertImage(ctx, image))
	image, found, err = s.datastore.GetImage(ctx, image.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(image.GetId(), results[0].ID)

	results, err = s.datastore.Search(ctx, pkgSearch.NewQueryBuilder().AddBools(pkgSearch.Fixable, true).ProtoQuery())
	s.NoError(err)
	s.Len(results, 0)
}

func (s *ImagePostgresDataStoreTestSuite) TestUpdateVulnStateWithPostgres() {
	image := fixtures.GetImageWithUniqueComponents()
	ctx := sac.WithAllAccess(context.Background())

	s.NoError(s.datastore.UpsertImage(ctx, image))
	_, found, err := s.datastore.GetImage(ctx, image.GetId())
	s.NoError(err)
	s.True(found)

	cloned := image.Clone()
	cloned.Id = "cloned"
	s.NoError(s.datastore.UpsertImage(ctx, cloned))
	_, found, err = s.datastore.GetImage(ctx, cloned.GetId())
	s.NoError(err)
	s.True(found)

	results, err := s.datastore.Search(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 0)

	var unsnoozedCVEs []string
	for _, component := range cloned.GetScan().GetComponents() {
		s.Require().GreaterOrEqual(len(component.GetVulns()), 2)
		err := s.datastore.UpdateVulnerabilityState(ctx, component.GetVulns()[0].GetCve(), []string{cloned.GetId()}, storage.VulnerabilityState_DEFERRED)
		s.NoError(err)
		unsnoozedCVEs = append(unsnoozedCVEs, component.GetVulns()[1].GetCve())

	}

	results, err = s.datastore.Search(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(cloned.GetId(), results[0].ID)

	// There are still some unsnoozed vulnerabilities in both images.
	results, err = s.datastore.Search(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 2)
	s.ElementsMatch([]string{image.GetId(), cloned.GetId()}, pkgSearch.ResultsToIDs(results))

	results, err = s.datastore.Search(ctx, pkgSearch.ConjunctionQuery(
		pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery(),
		pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, unsnoozedCVEs...).ProtoQuery(),
	))
	s.NoError(err)
	s.Len(results, 0)

	results, err = s.datastore.Search(ctx, pkgSearch.ConjunctionQuery(
		pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).ProtoQuery(),
		pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.CVE, unsnoozedCVEs...).ProtoQuery(),
	))
	s.NoError(err)
	s.Len(results, 2)
	s.ElementsMatch([]string{image.GetId(), cloned.GetId()}, pkgSearch.ResultsToIDs(results))
}

// Test sort by Component search label sorts by Component+Version to ensure backward compatibility.
func (s *ImagePostgresDataStoreTestSuite) TestSortByComponent() {
	ctx := sac.WithAllAccess(context.Background())
	node := fixtures.GetImageWithUniqueComponents()
	componentIDs := make([]string, 0, len(node.GetScan().GetComponents()))
	for _, component := range node.GetScan().GetComponents() {
		componentIDs = append(componentIDs,
			scancomponent.ComponentID(
				component.GetName(),
				component.GetVersion(),
				node.GetScan().GetOperatingSystem(),
			))
	}

	s.NoError(s.datastore.UpsertImage(ctx, node))

	// Verify sort by Component search label is transformed to sort by Component+Version.
	query := pkgSearch.EmptyQuery()
	query.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field: pkgSearch.Component.String(),
			},
		},
	}
	// Component ID is Name+Version+Operating System. Therefore, sort by ID is same as Component+Version.
	sort.SliceStable(componentIDs, func(i, j int) bool {
		return componentIDs[i] < componentIDs[j]
	})
	results, err := s.componentDataStore.Search(ctx, query)
	s.NoError(err)
	s.Equal(componentIDs, pkgSearch.ResultsToIDs(results))

	// Verify reverse sort.
	sort.SliceStable(componentIDs, func(i, j int) bool {
		return componentIDs[i] > componentIDs[j]
	})
	query.Pagination.SortOptions[0].Reversed = true
	results, err = s.componentDataStore.Search(ctx, query)
	s.NoError(err)
	s.Equal(componentIDs, pkgSearch.ResultsToIDs(results))
}
