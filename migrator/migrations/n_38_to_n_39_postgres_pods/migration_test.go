//go:build sql_integration

package n38ton39

import (
	"context"
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	legacy "github.com/stackrox/rox/migrator/migrations/n_38_to_n_39_postgres_pods/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_38_to_n_39_postgres_pods/postgres"
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

func (s *postgresMigrationSuite) TestPodMigration() {
	newStore := pgStore.New(s.postgresDB.DB)
	legacyStore, err := legacy.New(s.legacyDB)
	s.NoError(err)

	// Prepare data and write to legacy DB
	var pods []*storage.Pod
	countBadIDs := 0
	for i := 0; i < 200; i++ {
		pod := &storage.Pod{}
		s.NoError(testutils.FullInit(pod, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		if i%10 == 0 {
			pod.Id = strconv.Itoa(i)
			countBadIDs = countBadIDs + 1
		}
		pods = append(pods, pod)
	}

	s.NoError(legacyStore.UpsertMany(s.ctx, pods))

	// Move
	s.NoError(move(s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(pods)-countBadIDs, count)
	for _, pod := range pods {
		if pgutils.NilOrUUID(pod.GetId()) != nil {
			fetched, exists, err := newStore.Get(s.ctx, pod.GetId())
			s.NoError(err)
			s.True(exists)
			s.Equal(pod, fetched)
		}
	}
}
