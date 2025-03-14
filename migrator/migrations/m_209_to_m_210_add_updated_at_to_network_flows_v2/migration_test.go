//go:build sql_integration

package m209tom210

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_209_to_m_210_add_updated_at_to_network_flows_v2/test/schema/old"
	"github.com/stackrox/rox/migrator/migrations/m_209_to_m_210_add_updated_at_to_network_flows_v2/test/stores/previous"
	"github.com/stackrox/rox/migrator/migrations/m_209_to_m_210_add_updated_at_to_network_flows_v2/test/stores/updated"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	cluster1Count = 10
	cluster2Count = 15
)

var (
	cluster1 = uuid.NewV4().String()
	cluster2 = uuid.NewV4().String()
)

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context

	oldStore1 previous.FlowStore
	oldStore2 previous.FlowStore
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)

	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), old.CreateTableNetworkFlowsStmt)
	s.oldStore1 = previous.New(s.db.DB, cluster1)
	s.oldStore2 = previous.New(s.db.DB, cluster2)
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) TestMigration() {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()
	s.addFlows(ctx, s.oldStore1, cluster1, cluster1Count)
	s.addFlows(ctx, s.oldStore2, cluster2, cluster2Count)

	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}

	s.Require().NoError(migration.Run(dbs))

	newStore1 := updated.New(s.db.DB, cluster1)
	newStore2 := updated.New(s.db.DB, cluster2)

	flows1, _, err := newStore1.GetAllFlows(ctx, nil)
	s.Assert().NoError(err)
	s.Equal(cluster1Count, len(flows1))
	s.assertUpdatedAt(flows1)

	flows2, _, err := newStore2.GetAllFlows(ctx, nil)
	s.Assert().NoError(err)
	s.Equal(cluster2Count, len(flows2))
	s.assertUpdatedAt(flows2)
}

func (s *migrationTestSuite) addFlows(ctx context.Context, store previous.FlowStore, clusterID string, count int) {
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

func (s *migrationTestSuite) assertUpdatedAt(flows []*storage.NetworkFlow) {
	for _, flow := range flows {
		s.T().Log(flow)
		s.Assert().NotNil(flow.GetUpdatedAt())
	}
}
