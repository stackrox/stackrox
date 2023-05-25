//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

type VersionsStoreSuite struct {
	suite.Suite
	store  Store
	testDB *pgtest.TestPostgres
}

func TestVersionsStore(t *testing.T) {
	suite.Run(t, new(VersionsStoreSuite))
}

func (s *VersionsStoreSuite) SetupTest() {

	s.testDB = pgtest.ForT(s.T())
	s.store = New(s.testDB.DB)
}

func (s *VersionsStoreSuite) TearDownTest() {
	s.testDB.Teardown(s.T())
}

func (s *VersionsStoreSuite) TestStore() {
	ctx := sac.WithAllAccess(context.Background())

	store := s.store

	version := &storage.Version{}
	s.NoError(testutils.FullInit(version, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	version.LastPersisted = nil

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
	version.LastPersisted = nil
	s.NoError(store.Upsert(ctx, version))

	foundVersion, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal(version, foundVersion)

	version = &storage.Version{}
	s.NoError(testutils.FullInit(version, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	version.LastPersisted = nil
	s.NoError(store.Upsert(ctx, version))

	foundVersion, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal(version, foundVersion)
}
