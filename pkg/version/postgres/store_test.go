//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
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
	testDB      *pgtest.TestPostgres
}

func TestVersionsStore(t *testing.T) {
	suite.Run(t, new(VersionsStoreSuite))
}

func (s *VersionsStoreSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.testDB = pgtest.ForT(s.T())
	s.store = New(sac.WithAllAccess(context.Background()), s.testDB.Pool)
}

func (s *VersionsStoreSuite) TearDownTest() {
	s.testDB.Teardown(s.T())
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
