//go:build sql_integration

package n6ton7

import (
	"context"
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	legacy "github.com/stackrox/rox/migrator/migrations/n_06_to_n_07_postgres_alerts/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_06_to_n_07_postgres_alerts/postgres"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(postgresMigrationSuite))
}

type postgresMigrationSuite struct {
	suite.Suite
	ctx context.Context

	legacyDB   *rocksdb.RocksDB
	postgresDB *pghelper.TestPostgres
}

var _ suite.TearDownTestSuite = (*postgresMigrationSuite)(nil)

func (s *postgresMigrationSuite) SetupTest() {
	var err error
	s.legacyDB, err = rocksdb.NewTemp(s.T().Name())
	s.NoError(err)

	s.Require().NoError(err)

	s.ctx = sac.WithAllAccess(context.Background())
	s.postgresDB = pghelper.ForT(s.T(), false)
}

func (s *postgresMigrationSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.legacyDB)
	s.postgresDB.Teardown(s.T())
}

func (s *postgresMigrationSuite) TestAlertMigration() {
	newStore := pgStore.New(s.postgresDB.DB)
	legacyStore, err := legacy.New(s.legacyDB)
	s.NoError(err)

	// Prepare data and write to legacy DB
	var alerts []*storage.Alert
	countBadIDs := 0
	for i := 0; i < 200; i++ {
		alert := &storage.Alert{}
		s.NoError(testutils.FullInit(alert, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		if i%10 == 0 {
			alert.Id = strconv.Itoa(i)
			countBadIDs = countBadIDs + 1
		}
		alerts = append(alerts, alert)
	}

	s.NoError(legacyStore.UpsertMany(s.ctx, alerts))

	// Move
	s.NoError(move(s.ctx, s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(alerts)-countBadIDs, count)
	for _, alert := range alerts {
		if pgutils.NilOrUUID(alert.GetId()) != nil {
			fetched, exists, err := newStore.Get(s.ctx, alert.GetId())
			s.NoError(err)
			s.True(exists)
			s.Equal(alert, fetched)
		}
	}
}
