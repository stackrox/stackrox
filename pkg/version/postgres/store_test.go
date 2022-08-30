//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

type VersionsStoreSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
	store       Store
	pool        *pgxpool.Pool
}

func TestVersionsStore(t *testing.T) {
	suite.Run(t, new(VersionsStoreSuite))
}

func (s *VersionsStoreSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(ctx, config)
	s.Require().NoError(err)

	Destroy(ctx, pool)

	s.pool = pool
	s.store = New(ctx, pool)
}

func (s *VersionsStoreSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
	s.envIsolator.RestoreAll()
}

func (s *VersionsStoreSuite) TestStore() {
	ctx := sac.WithAllAccess(context.Background())

	store := s.store

	version := &storage.Version{}
	s.NoError(testutils.FullInit(version, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	foundVersion, exists, err := store.Get(ctx)
	s.NoError(err)
	s.False(exists)
	s.Nil(foundVersion)

	withNoAccessCtx := sac.WithNoAccess(ctx)

	s.NoError(store.Upsert(ctx, version))
	foundVersion, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal(version, foundVersion)

	foundVersion, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal(version, foundVersion)

	s.NoError(store.Delete(ctx))
	foundVersion, exists, err = store.Get(ctx)
	s.NoError(err)
	s.False(exists)
	s.Nil(foundVersion)

	s.ErrorIs(store.Delete(withNoAccessCtx), sac.ErrResourceAccessDenied)

	version = &storage.Version{}
	s.NoError(testutils.FullInit(version, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	s.NoError(store.Upsert(ctx, version))

	foundVersion, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal(version, foundVersion)

	version = &storage.Version{}
	s.NoError(testutils.FullInit(version, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	s.NoError(store.Upsert(ctx, version))

	foundVersion, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal(version, foundVersion)
}
