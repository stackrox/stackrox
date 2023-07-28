//go:build sql_integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

type UsageStoreSuite struct {
	suite.Suite
	store  Store
	testDB *pgtest.TestPostgres
}

func TestUsageStore(t *testing.T) {
	suite.Run(t, new(UsageStoreSuite))
}

func (s *UsageStoreSuite) SetupTest() {
	s.testDB = pgtest.ForT(s.T())
	s.store = New(s.testDB.DB)
}

func (s *UsageStoreSuite) TearDownTest() {
	s.testDB.Teardown(s.T())
}

func (s *UsageStoreSuite) TestStore() {
	ctx := sac.WithAllAccess(context.Background())

	store := s.store

	now := time.Now().Add(-time.Hour).UTC().Truncate(time.Microsecond)
	from := protoconv.ConvertTimeToTimestamp(now)
	to := protoconv.ConvertTimeToTimestamp(now.Add(10 * time.Minute))
	three := protoconv.ConvertTimeToTimestamp(now.Add(11 * time.Minute))

	records := []storage.Usage{
		{Timestamp: from},
		{Timestamp: to,
			NumNodes:    5,
			NumCpuUnits: 50,
		},
	}

	foundRecords, err := store.Get(ctx, from, to)
	s.Require().NoError(err)
	s.Empty(foundRecords)

	s.Require().NoError(store.Insert(ctx, &records[0]))
	s.Require().NoError(store.Insert(ctx, &records[1]))
	foundRecords, err = store.Get(ctx, from, to)
	s.Require().NoError(err)
	s.Require().Len(foundRecords, 1)
	s.Equal(records[0], foundRecords[0])

	foundRecords, err = store.Get(ctx, to, three)
	s.Require().NoError(err)
	s.Require().Len(foundRecords, 1)
	s.Equal(records[1], foundRecords[0])

	foundRecords, err = store.Get(ctx, from, three)
	s.Require().NoError(err)
	s.Require().Len(foundRecords, 2)
	s.Equal(records, foundRecords)

	foundRecords, err = store.Get(ctx, nil, nil)
	s.Require().NoError(err)
	s.Require().Len(foundRecords, 2)
	s.Equal(records, foundRecords)
}

func (s *UsageStoreSuite) TestGet() {
	ctx := sac.WithAllAccess(context.Background())

	store := s.store

	now := time.Now().UTC().Truncate(time.Microsecond)
	from := protoconv.ConvertTimeToTimestamp(now)
	to := protoconv.ConvertTimeToTimestamp(now.Add(10 * time.Minute))

	records := []storage.Usage{
		{Timestamp: from},
		{Timestamp: to,
			NumNodes:    5,
			NumCpuUnits: 50,
		},
	}

	s.Require().NoError(store.Insert(ctx, &records[0]))
	foundRecords, err := store.Get(ctx, from, to)
	s.Require().NoError(err)
	s.Require().Len(foundRecords, 1)
	s.Equal(records[0], foundRecords[0])
}
