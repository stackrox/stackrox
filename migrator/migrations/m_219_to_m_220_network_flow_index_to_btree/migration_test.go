//go:build sql_integration

package m219tom220

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/indexhelper"
	"github.com/stackrox/rox/migrator/migrations/m_219_to_m_220_network_flow_index_to_btree/test/schema"
	"github.com/stackrox/rox/migrator/migrations/m_219_to_m_220_network_flow_index_to_btree/test/store"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stretchr/testify/suite"
)

const (
	cluster1Count = 100000
	cluster2Count = 150000
)

var (
	cluster1 = fixtureconsts.Cluster1
	cluster2 = fixtureconsts.Cluster2
)

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context

	oldStore1 store.FlowStore
	oldStore2 store.FlowStore
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)

	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), schema.CreateTableNetworkFlowsStmt)
	s.oldStore1 = store.New(s.db.DB, cluster1)
	s.oldStore2 = store.New(s.db.DB, cluster2)
}

func (s *migrationTestSuite) TestMigration() {
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.addFlows(s.ctx, s.oldStore1, cluster1, cluster1Count)
	s.addFlows(s.ctx, s.oldStore2, cluster2, cluster2Count)

	// Verify hash indexes
	exists, err := indexhelper.IndexExists(s.ctx, s.db, tableName, indexSrcEntity, "hash")
	s.Require().NoError(err)
	s.Require().True(exists)
	exists, err = indexhelper.IndexExists(s.ctx, s.db, tableName, indexDstEntity, "hash")
	s.Require().NoError(err)
	s.Require().True(exists)

	s.Assert().NoError(migration.Run(dbs))

	s.verifyNewIndexes()

	// Run the migration a second time to ensure idempotentcy.
	s.Assert().NoError(migration.Run(dbs))

	s.verifyNewIndexes()
}

func (s *migrationTestSuite) verifyNewIndexes() {
	// Verify hash indexes no longer exist.
	exists, err := indexhelper.IndexExists(s.ctx, s.db, tableName, indexSrcEntity, "hash")
	s.Assert().NoError(err)
	s.Assert().False(exists)
	exists, err = indexhelper.IndexExists(s.ctx, s.db, tableName, indexDstEntity, "hash")
	s.Assert().NoError(err)
	s.Assert().False(exists)

	exists, err = indexhelper.IndexExists(s.ctx, s.db, tableName, indexSrcEntity, "btree")
	s.Assert().NoError(err)
	s.Assert().True(exists)
	exists, err = indexhelper.IndexExists(s.ctx, s.db, tableName, indexDstEntity, "btree")
	s.Assert().NoError(err)
	s.Assert().True(exists)
}

func (s *migrationTestSuite) addFlows(ctx context.Context, store store.FlowStore, clusterID string, count int) {
	flows := make([]*storage.NetworkFlow, 0, count)
	zeroTs := timestamp.MicroTS(0)

	for i := 0; i < count; i++ {
		flow := &storage.NetworkFlow{}
		s.NoError(testutils.FullInit(flow, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		flow.ClusterId = clusterID
		flows = append(flows, flow)
	}
	s.NoError(store.UpsertFlows(ctx, flows, zeroTs))
}
