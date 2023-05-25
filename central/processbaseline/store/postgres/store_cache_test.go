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

type ProcessBaselinesCacheSuite struct {
	suite.Suite
	testDB *pgtest.TestPostgres
}

func TestProcessBaselinesCacheStore(t *testing.T) {
	suite.Run(t, new(ProcessBaselinesCacheSuite))
}

func (s *ProcessBaselinesCacheSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
}

func (s *ProcessBaselinesCacheSuite) SetupTest() {
	ctx := sac.WithAllAccess(context.Background())
	tag, err := s.testDB.Exec(ctx, "TRUNCATE process_baselines CASCADE")
	s.T().Log("process_baselines", tag)
	s.NoError(err)
}

func (s *ProcessBaselinesCacheSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *ProcessBaselinesCacheSuite) TestCacheStore() {
	dbStore := New(s.testDB.DB)

	store, err := NewWithCache(dbStore)
	s.NoError(err)
	ctx := sac.WithAllAccess(context.Background())

	processBaseline := &storage.ProcessBaseline{}
	s.NoError(testutils.FullInit(processBaseline, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	foundProcessBaseline, exists, err := store.Get(ctx, processBaseline.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundProcessBaseline)

	s.NoError(store.Upsert(ctx, processBaseline))
	foundProcessBaseline, exists, err = store.Get(ctx, processBaseline.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(processBaseline, foundProcessBaseline)

	processBaselineCount, err := store.Count(ctx)
	s.NoError(err)
	s.Equal(1, processBaselineCount)

	processBaselineExists, err := store.Exists(ctx, processBaseline.GetId())
	s.NoError(err)
	s.True(processBaselineExists)
	s.NoError(store.Upsert(ctx, processBaseline))

	foundProcessBaseline, exists, err = store.Get(ctx, processBaseline.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(processBaseline, foundProcessBaseline)

	s.NoError(store.Delete(ctx, processBaseline.GetId()))
	foundProcessBaseline, exists, err = store.Get(ctx, processBaseline.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundProcessBaseline)

	var processBaselines []*storage.ProcessBaseline
	var processBaselineIDs []string
	for i := 0; i < 200; i++ {
		processBaseline := &storage.ProcessBaseline{}
		s.NoError(testutils.FullInit(processBaseline, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		processBaselines = append(processBaselines, processBaseline)
		processBaselineIDs = append(processBaselineIDs, processBaseline.GetId())
	}

	s.NoError(store.UpsertMany(ctx, processBaselines))

	processBaselineCount, err = store.Count(ctx)
	s.NoError(err)
	s.Equal(200, processBaselineCount)

	s.NoError(store.DeleteMany(ctx, processBaselineIDs))

	processBaselineCount, err = store.Count(ctx)
	s.NoError(err)
	s.Equal(0, processBaselineCount)
}
