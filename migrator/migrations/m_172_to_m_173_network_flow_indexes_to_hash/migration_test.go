//go:build sql_integration

package m172tom173

import (
	"context"
	"testing"

	oldSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	newSchema "github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_network_flow_indexes_to_hash/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/suite"
)

type categoriesMigrationTestSuite struct {
	suite.Suite

	db *pghelper.TestPostgres
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(categoriesMigrationTestSuite))
}

func (s *categoriesMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), true)
	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), oldSchema.CreateTableNetworkFlowsStmt)
}

func (s *categoriesMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *categoriesMigrationTestSuite) TestMigration() {
	var indexSet set.StringSet
	indexSet.Add("network_flows_pkey")
	indexSet.Add("network_flows_cluster")
	indexSet.Add("network_flows_dst")
	indexSet.Add("network_flows_lastseentimestamp")
	indexSet.Add("network_flows_src")

	dbs := &types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.db.GetGormDB(),
	}

	indexes, err := s.db.GetGormDB().Migrator().GetIndexes(&oldSchema.NetworkFlows{})
	s.NoError(err)

	for _, index := range indexes {
		s.True(indexSet.Contains(index.Name()))
	}

	s.NoError(migration.Run(dbs))

	indexes, err = s.db.GetGormDB().Migrator().GetIndexes(&newSchema.NetworkFlows{})
	s.NoError(err)

	for _, index := range indexes {
		s.True(indexSet.Contains(index.Name()))
	}
}
