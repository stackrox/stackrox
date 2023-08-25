//go:build sql_integration

package n10ton11

import (
	"context"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	legacy "github.com/stackrox/rox/migrator/migrations/n_10_to_n_11_postgres_cluster_health_statuses/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_10_to_n_11_postgres_cluster_health_statuses/postgres"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
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

func (s *postgresMigrationSuite) TestClusterHealthStatusMigration() {
	newStore := pgStore.New(s.postgresDB.DB)
	legacyStore, err := legacy.New(s.legacyDB)
	s.NoError(err)

	// Prepare data and write to legacy DB
	var clusterHealthStatuses []*storage.ClusterHealthStatus
	countBadIDs := 0
	for i := 0; i < 200; i++ {
		clusterHealthStatus := &storage.ClusterHealthStatus{}
		s.NoError(testutils.FullInit(clusterHealthStatus, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		if i%10 == 0 {
			clusterHealthStatus.Id = strconv.Itoa(i)
			countBadIDs = countBadIDs + 1
		}
		clusterHealthStatuses = append(clusterHealthStatuses, clusterHealthStatus)
	}

	s.NoError(legacyStore.UpsertMany(s.ctx, clusterHealthStatuses))

	// Move
	s.NoError(move(s.ctx, s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(clusterHealthStatuses)-countBadIDs, count)
	for _, clusterHealthStatus := range clusterHealthStatuses {
		if pgutils.NilOrUUID(clusterHealthStatus.GetId()) != nil {
			fetched, exists, err := newStore.Get(s.ctx, clusterHealthStatus.GetId())
			s.NoError(err)
			s.True(exists)
			s.Equal(clusterHealthStatus, fetched)
		}
	}
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.ClusterHealthStatus).GetId())
}

func (s *postgresMigrationSuite) TestClusterHealthStatusMigrationWithBadData() {
	newStore := pgStore.New(s.postgresDB.DB)
	legacyStore, err := legacy.New(s.legacyDB)
	s.NoError(err)

	baseCRUD := generic.NewCRUD(s.legacyDB, []byte("clusters_health_status"), keyFunc, nil, false)

	// Prepare data and write to legacy DB
	var clusterHealthStatuss []*storage.ClusterHealthStatus
	for i := 0; i < 200; i++ {
		clusterHealthStatus := &storage.ClusterHealthStatus{}
		s.NoError(testutils.FullInit(clusterHealthStatus, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		clusterHealthStatuss = append(clusterHealthStatuss, clusterHealthStatus)
	}

	for _, chs := range clusterHealthStatuss {
		id := chs.GetId()
		cloned := chs.Clone()
		cloned.Id = ""
		s.NoError(baseCRUD.UpsertWithID(id, cloned))
	}

	// Move
	s.NoError(move(s.ctx, s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(len(clusterHealthStatuss), count)
	for _, clusterHealthStatus := range clusterHealthStatuss {
		fetched, exists, err := newStore.Get(s.ctx, clusterHealthStatus.GetId())
		s.NoError(err)
		s.True(exists)
		s.Equal(clusterHealthStatus, fetched)
	}
}
