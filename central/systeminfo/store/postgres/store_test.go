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

type SystemInfosStoreSuite struct {
	suite.Suite
	store  Store
	testDB *pgtest.TestPostgres
}

func TestSystemInfosStore(t *testing.T) {
	suite.Run(t, new(SystemInfosStoreSuite))
}

func (s *SystemInfosStoreSuite) SetupTest() {

	s.testDB = pgtest.ForT(s.T())
	s.store = New(s.testDB.DB)
}

func (s *SystemInfosStoreSuite) TearDownTest() {
	s.testDB.Teardown(s.T())
}

func (s *SystemInfosStoreSuite) TestStore() {
	ctx := sac.WithAllAccess(context.Background())

	store := s.store

	systemInfo := &storage.SystemInfo{}
	s.NoError(testutils.FullInit(systemInfo, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	foundSystemInfo, exists, err := store.Get(ctx)
	s.NoError(err)
	s.False(exists)
	s.Nil(foundSystemInfo)

	withNoAccessCtx := sac.WithNoAccess(ctx)

	s.NoError(store.Upsert(ctx, systemInfo))
	foundSystemInfo, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal(systemInfo, foundSystemInfo)

	foundSystemInfo, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal(systemInfo, foundSystemInfo)

	s.NoError(store.Delete(ctx))
	foundSystemInfo, exists, err = store.Get(ctx)
	s.NoError(err)
	s.False(exists)
	s.Nil(foundSystemInfo)

	s.ErrorIs(store.Delete(withNoAccessCtx), sac.ErrResourceAccessDenied)

	systemInfo = &storage.SystemInfo{}
	s.NoError(testutils.FullInit(systemInfo, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	s.NoError(store.Upsert(ctx, systemInfo))

	foundSystemInfo, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal(systemInfo, foundSystemInfo)

	systemInfo = &storage.SystemInfo{}
	s.NoError(testutils.FullInit(systemInfo, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	s.NoError(store.Upsert(ctx, systemInfo))

	foundSystemInfo, exists, err = store.Get(ctx)
	s.NoError(err)
	s.True(exists)
	s.Equal(systemInfo, foundSystemInfo)
}
