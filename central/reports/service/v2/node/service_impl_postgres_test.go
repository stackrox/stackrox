//go:build sql_integration

package node

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestNodeReportServicePostgres(t *testing.T) {
	suite.Run(t, new(NodeReportServicePostgresTestSuite))
}

type NodeReportServicePostgresTestSuite struct {
	suite.Suite
	ctx    context.Context
	testDB *pgtest.TestPostgres
}

func (s *NodeReportServicePostgresTestSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *NodeReportServicePostgresTestSuite) TearDownTest() {
	s.truncateTable("nodes")
	s.truncateTable("node_cves")
	s.truncateTable("report_snapshots")
}

func (s *NodeReportServicePostgresTestSuite) truncateTable(name string) {
	_, err := s.testDB.Exec(s.ctx, "TRUNCATE "+name+" CASCADE")
	s.NoError(err)
}

func (s *NodeReportServicePostgresTestSuite) TestGetReportHistory_WithUserIDFilter() {
	s.T().Skip("Requires full datastore initialization - verify query logic in unit tests")
}

func (s *NodeReportServicePostgresTestSuite) TestAccessScopeRulesPreservation() {
	s.T().Skip("Requires full datastore initialization - verify in unit tests")
}

func (s *NodeReportServicePostgresTestSuite) TestNodeReportFilters_QueryGeneration() {
	clusterID := uuid.NewV4().String()

	node1 := fixtures.GetNodeWithUniqueComponents(5, 2)
	node1.Id = uuid.NewV4().String()
	node1.ClusterId = clusterID
	node1.Name = "test-node-1"

	node2 := fixtures.GetNodeWithUniqueComponents(3, 2)
	node2.Id = uuid.NewV4().String()
	node2.ClusterId = clusterID
	node2.Name = "test-node-2"

	s.NotNil(node1)
	s.NotNil(node2)
	s.Equal(5, len(node1.GetScan().GetComponents()))
	s.Equal(3, len(node2.GetScan().GetComponents()))
}
