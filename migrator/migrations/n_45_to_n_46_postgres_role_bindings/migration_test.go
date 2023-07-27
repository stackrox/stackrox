//go:build sql_integration

package n45ton46

import (
	"context"
	"strconv"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	legacy "github.com/stackrox/rox/migrator/migrations/n_45_to_n_46_postgres_role_bindings/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_45_to_n_46_postgres_role_bindings/postgres"
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

func (s *postgresMigrationSuite) TestK8SRoleBindingMigration() {
	newStore := pgStore.New(s.postgresDB.DB)
	legacyStore, err := legacy.New(s.legacyDB)
	s.NoError(err)

	// Prepare data and write to legacy DB
	var k8SRoleBindings []*storage.K8SRoleBinding
	countBadIDs := 0
	for i := 0; i < 200; i++ {
		k8SRoleBinding := &storage.K8SRoleBinding{}
		s.NoError(testutils.FullInit(k8SRoleBinding, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		if i%10 == 0 {
			k8SRoleBinding.Id = strconv.Itoa(i)
			countBadIDs = countBadIDs + 1
		}
		k8SRoleBindings = append(k8SRoleBindings, k8SRoleBinding)
	}

	s.NoError(legacyStore.UpsertMany(s.ctx, k8SRoleBindings))

	// Move
	s.NoError(move(s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(k8SRoleBindings)-countBadIDs, count)
	for _, k8SRoleBinding := range k8SRoleBindings {
		if pgutils.NilOrUUID(k8SRoleBinding.GetId()) != nil {
			fetched, exists, err := newStore.Get(s.ctx, k8SRoleBinding.GetId())
			s.NoError(err)
			s.True(exists)
			s.Equal(k8SRoleBinding, fetched)
		}
	}
}
