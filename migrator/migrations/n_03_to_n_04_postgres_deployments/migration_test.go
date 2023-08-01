//go:build sql_integration

package n3ton4

import (
	"context"
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	legacy "github.com/stackrox/rox/migrator/migrations/n_03_to_n_04_postgres_deployments/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_03_to_n_04_postgres_deployments/postgres"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
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

func (s *postgresMigrationSuite) TestDeploymentMigration() {
	newStore := pgStore.New(s.postgresDB.DB)
	dacky, err := dackbox.NewRocksDBDackBox(s.legacyDB, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	s.NoError(err)
	legacyStore := legacy.New(dacky, concurrency.NewKeyFence())

	// Prepare data and write to legacy DB
	var deployments []*storage.Deployment
	countBadIDs := 0
	for i := 0; i < 200; i++ {
		deployment := &storage.Deployment{}
		s.NoError(testutils.FullInit(deployment, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		if i%10 == 0 {
			deployment.Id = strconv.Itoa(i)
			countBadIDs = countBadIDs + 1
		}
		deployments = append(deployments, deployment)
	}

	s.NoError(legacyStore.UpsertMany(s.ctx, deployments))

	// Move
	s.NoError(move(s.ctx, s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(deployments)-countBadIDs, count)
	for _, deployment := range deployments {
		if pgutils.NilOrUUID(deployment.GetId()) != nil {
			fetched, exists, err := newStore.Get(s.ctx, deployment.GetId())
			s.NoError(err)
			s.True(exists)
			s.Equal(deployment, fetched)
		}
	}
}
