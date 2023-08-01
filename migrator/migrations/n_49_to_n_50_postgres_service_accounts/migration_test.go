//go:build sql_integration

package n49ton50

import (
	"context"
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	legacy "github.com/stackrox/rox/migrator/migrations/n_49_to_n_50_postgres_service_accounts/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_49_to_n_50_postgres_service_accounts/postgres"
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

func (s *postgresMigrationSuite) TestServiceAccountMigration() {
	newStore := pgStore.New(s.postgresDB.DB)
	legacyStore, err := legacy.New(s.legacyDB)
	s.NoError(err)

	// Prepare data and write to legacy DB
	var serviceAccounts []*storage.ServiceAccount
	countBadIDs := 0
	for i := 0; i < 200; i++ {
		serviceAccount := &storage.ServiceAccount{}
		s.NoError(testutils.FullInit(serviceAccount, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		if i%10 == 0 {
			serviceAccount.Id = strconv.Itoa(i)
			countBadIDs = countBadIDs + 1
		}
		serviceAccounts = append(serviceAccounts, serviceAccount)
	}

	s.NoError(legacyStore.UpsertMany(s.ctx, serviceAccounts))

	// Move
	s.NoError(move(s.ctx, s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(serviceAccounts)-countBadIDs, count)
	for _, serviceAccount := range serviceAccounts {
		if pgutils.NilOrUUID(serviceAccount.GetId()) != nil {
			fetched, exists, err := newStore.Get(s.ctx, serviceAccount.GetId())
			s.NoError(err)
			s.True(exists)
			s.Equal(serviceAccount, fetched)
		}
	}
}
