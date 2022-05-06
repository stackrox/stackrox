//go:build sql_integration
// +build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/image/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/postgres/schema"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

func TestImageDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(ImagePostgresDataStoreTestSuite))
}

type ImagePostgresDataStoreTestSuite struct {
	suite.Suite

	ctx       context.Context
	db        *pgxpool.Pool
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
	s.db = pool

	postgres.Destroy(s.ctx, s.db)

	s.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(s.T()))
	s.datastore = NewWithPostgres(postgres.New(s.ctx, s.db, false), postgres.NewIndexer(s.db), s.mockRisk, ranking.ImageRanker(), ranking.ComponentRanker())
}

func (s *ImagePostgresDataStoreTestSuite) TearDownSuite() {
	s.envIsolator.RestoreAll()
	s.db.Close()
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
