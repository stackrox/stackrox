//go:build sql_integration

package n28ton29

import (
	"context"
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	legacy "github.com/stackrox/rox/migrator/migrations/n_28_to_n_29_postgres_network_baselines/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_28_to_n_29_postgres_network_baselines/postgres"
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

func (s *postgresMigrationSuite) TestNetworkBaselineMigration() {
	newStore := pgStore.New(s.postgresDB.DB)
	legacyStore, err := legacy.New(s.legacyDB)
	s.NoError(err)

	// Prepare data and write to legacy DB
	var networkBaselines []*storage.NetworkBaseline
	countBadIDs := 0
	for i := 0; i < 200; i++ {
		networkBaseline := &storage.NetworkBaseline{}
		s.NoError(testutils.FullInit(networkBaseline, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		if i%10 == 0 {
			networkBaseline.DeploymentId = strconv.Itoa(i)
			countBadIDs = countBadIDs + 1
		}
		networkBaselines = append(networkBaselines, networkBaseline)
	}

	s.NoError(legacyStore.UpsertMany(s.ctx, networkBaselines))

	// Move
	s.NoError(move(s.ctx, s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(networkBaselines)-countBadIDs, count)
	for _, networkBaseline := range networkBaselines {
		if pgutils.NilOrUUID(networkBaseline.GetDeploymentId()) != nil {
			fetched, exists, err := newStore.Get(s.ctx, networkBaseline.GetDeploymentId())
			s.NoError(err)
			s.True(exists)
			s.Equal(networkBaseline, fetched)
		}
	}
}
