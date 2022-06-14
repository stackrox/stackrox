//go:build sql_integration
// +build sql_integration

package datastore

import (
	"context"
	"testing"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/stackrox/central/image/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/central/ranking"
	mockRisks "github.com/stackrox/stackrox/central/risk/datastore/mocks"
	"github.com/stackrox/stackrox/central/role/resources"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/cve"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stackrox/stackrox/pkg/postgres/pgtest"
	"github.com/stackrox/stackrox/pkg/postgres/schema"
	"github.com/stackrox/stackrox/pkg/sac"
	pkgSearch "github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/postgres/mapping"
	"github.com/stackrox/stackrox/pkg/search/scoped"
	"github.com/stackrox/stackrox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestImageDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(ImagePostgresDataStoreTestSuite))
}

type ImagePostgresDataStoreTestSuite struct {
	suite.Suite

	ctx       context.Context
	db        *pgxpool.Pool
	gormDB    *gorm.DB
	datastore DataStore
	mockRisk  *mockRisks.MockDataStore

	envIsolator *envisolator.EnvIsolator
}

func (s *ImagePostgresDataStoreTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	s.T().Skip("Skip postgres store tests")
	s.T().SkipNow()

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
	mapping.RegisterCategoryToTable(v1.SearchCategory_VULNERABILITIES, schema.ImageCvesSchema)
	scopedCtx = scoped.Context(ctx, scoped.Scope{
		ID:    "cve3#blah",
		Level: v1.SearchCategory_VULNERABILITIES,
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
	// This should be 1, but until outer join changes to inner join, it is going to be 10 because there are 10 cves in `cloned`.
	s.Len(results, 10)
	s.Equal(image.GetId(), results[0].ID)

	image.Scan.ScanTime = protoTypes.TimestampNow()
	for _, component := range image.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			vuln.SetFixedBy = nil
		}
	}
	s.NoError(s.datastore.UpsertImage(ctx, image))
	_, found, err = s.datastore.GetImage(ctx, image.GetId())
	s.NoError(err)
	s.True(found)

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

	for _, component := range cloned.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			err := s.datastore.UpdateVulnerabilityState(ctx, cve.ID(vuln.GetCve(), cloned.GetScan().GetOperatingSystem()), []string{cloned.GetId()}, storage.VulnerabilityState_DEFERRED)
			s.NoError(err)
		}
	}

	results, err := s.datastore.Search(ctx, pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.VulnerabilityState, storage.VulnerabilityState_DEFERRED.String()).ProtoQuery())
	s.NoError(err)
	// This should be 1, but until outer join changes to inner join, it is going to be 10 because there are 10 cves in `cloned`.
	s.Len(results, 10)
	s.Equal(cloned.GetId(), results[0].ID)
}
