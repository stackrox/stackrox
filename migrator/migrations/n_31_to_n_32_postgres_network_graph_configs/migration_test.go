//go:build sql_integration

package n31ton32

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	legacy "github.com/stackrox/rox/migrator/migrations/n_31_to_n_32_postgres_network_graph_configs/legacy"
	pgStore "github.com/stackrox/rox/migrator/migrations/n_31_to_n_32_postgres_network_graph_configs/postgres"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
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

func (s *postgresMigrationSuite) TestNetworkGraphConfigMigration() {
	newStore := pgStore.New(s.postgresDB.DB)
	legacyStore, err := legacy.New(s.legacyDB)
	s.NoError(err)

	// Prepare data and write to legacy DB

	networkGraphConfig := &storage.NetworkGraphConfig{}
	s.NoError(testutils.FullInit(networkGraphConfig, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	networkGraphConfig.Id = networkGraphConfigKey
	s.NoError(legacyStore.UpsertMany(s.ctx, []*storage.NetworkGraphConfig{networkGraphConfig}))

	// Move
	s.NoError(move(s.ctx, s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(1, count)

	fetched, exists, err := newStore.Get(s.ctx, networkGraphConfig.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(networkGraphConfig, fetched)
}

func (s *postgresMigrationSuite) TestNetworkGraphConfigMigrationWithEmptyID() {
	newStore := pgStore.New(s.postgresDB.DB)
	legacyStore, err := legacy.New(s.legacyDB)
	s.NoError(err)

	crud := generic.NewCRUD(s.legacyDB, []byte("networkgraphconfig"), nil, nil, false)
	networkGraphConfig := &storage.NetworkGraphConfig{}
	s.NoError(testutils.FullInit(networkGraphConfig, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	networkGraphConfig.Id = ""

	s.NoError(crud.UpsertWithID(networkGraphConfigKey, networkGraphConfig))

	// Move
	s.NoError(move(s.ctx, s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(1, count)

	networkGraphConfig.Id = networkGraphConfigKey
	fetched, exists, err := newStore.Get(s.ctx, networkGraphConfig.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(networkGraphConfig, fetched)
}

func (s *postgresMigrationSuite) TestNetworkGraphConfigMigrationMultiple() {
	newStore := pgStore.New(s.postgresDB.DB)
	legacyStore, err := legacy.New(s.legacyDB)
	s.NoError(err)

	crud := generic.NewCRUD(s.legacyDB, []byte("networkgraphconfig"), nil, nil, false)
	networkGraphConfig := &storage.NetworkGraphConfig{}
	s.NoError(testutils.FullInit(networkGraphConfig, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	networkGraphConfig.Id = ""
	s.NoError(crud.UpsertWithID(networkGraphConfigKey, networkGraphConfig))
	s.NoError(crud.UpsertWithID("random", networkGraphConfig))

	// Move
	s.NoError(move(s.ctx, s.postgresDB.GetGormDB(), s.postgresDB.DB, legacyStore))

	// Verify
	count, err := newStore.Count(s.ctx)
	s.NoError(err)
	s.Equal(1, count)

	networkGraphConfig.Id = networkGraphConfigKey
	fetched, exists, err := newStore.Get(s.ctx, networkGraphConfig.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(networkGraphConfigKey, fetched.GetId())
}
