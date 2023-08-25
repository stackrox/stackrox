//go:build sql_integration

package m172tom173

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	oldSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	"github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_network_flows_partition/stores/previous"
	"github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_network_flows_partition/stores/updated"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
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

type networkFlowsMigrationTestSuite struct {
	suite.Suite

	db        *pghelper.TestPostgres
	oldStore1 previous.FlowStore
	oldStore2 previous.FlowStore
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(networkFlowsMigrationTestSuite))
}

func (s *networkFlowsMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), false)
	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), oldSchema.CreateTableNetworkFlowsStmt)
	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), oldSchema.CreateTableClustersStmt)

	s.oldStore1 = previous.New(s.db.DB, cluster1)
	s.oldStore2 = previous.New(s.db.DB, cluster2)
}

func (s *networkFlowsMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *networkFlowsMigrationTestSuite) TestMigration() {
	// Add some data to the original tables via the old stores.
	s.addSomeOldData()

	_, err := s.db.DB.Exec(context.Background(), "insert into clusters (id) values ($1)", cluster1)
	s.NoError(err)
	_, err = s.db.DB.Exec(context.Background(), "insert into clusters (id) values ($1)", cluster2)
	s.NoError(err)

	dbs := &types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.db.GetGormDB(),
	}

	s.NoError(migration.Run(dbs))

	newStore1 := updated.New(s.db.DB, cluster1)
	newStore2 := updated.New(s.db.DB, cluster2)

	flows1, _, err := newStore1.GetAllFlows(context.Background(), nil)
	s.NoError(err)
	s.Equal(cluster1Count, len(flows1))

	flows2, _, err := newStore2.GetAllFlows(context.Background(), nil)
	s.NoError(err)
	s.Equal(cluster2Count, len(flows2))

}

func (s *networkFlowsMigrationTestSuite) addSomeOldData() {
	var networkFlows []*storage.NetworkFlow
	zeroTs := timestamp.MicroTS(0)

	// Add some to cluster 1
	for i := 0; i < cluster1Count; i++ {
		networkFlow := &storage.NetworkFlow{}
		s.NoError(testutils.FullInit(networkFlow, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		networkFlow.ClusterId = cluster1
		networkFlows = append(networkFlows, networkFlow)
	}
	s.NoError(s.oldStore1.UpsertFlows(context.Background(), networkFlows, zeroTs))

	networkFlows = nil
	// Add some to cluster 2
	for i := 0; i < cluster2Count; i++ {
		networkFlow := &storage.NetworkFlow{}
		s.NoError(testutils.FullInit(networkFlow, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		networkFlow.ClusterId = cluster2
		networkFlows = append(networkFlows, networkFlow)
	}
	s.NoError(s.oldStore2.UpsertFlows(context.Background(), networkFlows, zeroTs))
}
