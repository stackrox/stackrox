//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

type ImagesStoreSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func TestImagesStore(t *testing.T) {
	suite.Run(t, new(ImagesStoreSuite))
}

func (s *ImagesStoreSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}
}

func (s *ImagesStoreSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *ImagesStoreSuite) TestStore() {
	ctx := context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(ctx, config)
	s.NoError(err)
	defer pool.Close()

	Destroy(ctx, pool)

	gormDB := pgtest.OpenGormDB(s.T(), source)
	defer pgtest.CloseGormDB(s.T(), gormDB)
	store := CreateTableAndNewStore(ctx, pool, gormDB, false)

	image := fixtures.GetImage()
	s.NoError(testutils.FullInit(image, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	foundImage, exists, err := store.Get(ctx, image.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundImage)

	s.NoError(store.Upsert(ctx, image))
	foundImage, exists, err = store.Get(ctx, image.GetId())
	s.NoError(err)
	s.True(exists)
	cloned := image.Clone()
	for _, component := range cloned.GetScan().GetComponents() {
		for _, vuln := range component.GetVulns() {
			vuln.FirstSystemOccurrence = foundImage.GetLastUpdated()
			vuln.FirstImageOccurrence = foundImage.GetLastUpdated()
		}
	}
	s.Equal(cloned, foundImage)

	imageCount, err := store.Count(ctx)
	s.NoError(err)
	s.Equal(imageCount, 1)

	imageExists, err := store.Exists(ctx, image.GetId())
	s.NoError(err)
	s.True(imageExists)
	s.NoError(store.Upsert(ctx, image))

	foundImage, exists, err = store.Get(ctx, image.GetId())
	s.NoError(err)
	s.True(exists)

	// Reconcile the timestamps that are set during upsert.
	cloned.LastUpdated = foundImage.LastUpdated
	s.Equal(cloned, foundImage)

	s.NoError(store.Delete(ctx, image.GetId()))
	foundImage, exists, err = store.Get(ctx, image.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundImage)
}
