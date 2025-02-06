//go:build sql_integration

package postgres

import (
	"context"
	"testing"
	"time"

	nodeCveStore "github.com/stackrox/rox/central/cve/node/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type NodesStoreSuite struct {
	suite.Suite
	ctx    context.Context
	pool   postgres.DB
	gormDB *gorm.DB
}

func TestNodesStore(t *testing.T) {
	suite.Run(t, new(NodesStoreSuite))
}

func (s *NodesStoreSuite) SetupTest() {

	s.ctx = sac.WithAllAccess(context.Background())
	source := pgtest.GetConnectionString(s.T())

	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	s.pool, err = postgres.New(s.ctx, config)
	s.NoError(err)
	Destroy(s.ctx, s.pool)

	s.gormDB = pgtest.OpenGormDB(s.T(), source)
}

func (s *NodesStoreSuite) TearDownTest() {
	if s.pool != nil {
		s.pool.Close()
	}
	if s.gormDB != nil {
		pgtest.CloseGormDB(s.T(), s.gormDB)
	}
}

func (s *NodesStoreSuite) TestStore() {
	store := CreateTableAndNewStore(s.ctx, s.T(), s.pool, s.gormDB, false)

	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
	for _, comp := range node.GetScan().GetComponents() {
		comp.Vulns = nil
	}

	foundNode, exists, err := store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundNode)

	s.NoError(store.Upsert(s.ctx, node))
	foundNode, exists, err = store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	cloned := node.CloneVT()

	for _, component := range cloned.GetScan().GetComponents() {
		for _, vuln := range component.GetVulnerabilities() {
			vuln.CveBaseInfo.CreatedAt = node.GetLastUpdated()
		}
	}
	protoassert.Equal(s.T(), cloned, foundNode)

	nodeCount, err := store.Count(s.ctx, search.EmptyQuery())
	s.NoError(err)
	s.Equal(nodeCount, 1)

	nodeExists, err := store.Exists(s.ctx, node.GetId())
	s.NoError(err)
	s.True(nodeExists)
	s.NoError(store.Upsert(s.ctx, node))

	foundNode, exists, err = store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)

	// Reconcile the timestamps that are set during upsert.
	cloned.LastUpdated = foundNode.LastUpdated
	protoassert.Equal(s.T(), cloned, foundNode)

	s.NoError(store.Delete(s.ctx, node.GetId()))
	foundNode, exists, err = store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundNode)
}

func (s *NodesStoreSuite) TestStore_UpsertWithoutScan() {
	store := CreateTableAndNewStore(s.ctx, s.T(), s.pool, s.gormDB, false)

	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	foundNode, exists, err := store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundNode)

	s.NoError(store.Upsert(s.ctx, node))

	foundNode, exists, err = store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	s.NotNil(foundNode.GetScan())

	node = foundNode.CloneVT()
	node.Scan = nil
	s.NoError(store.Upsert(s.ctx, node))

	newNode, exists, err := store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)

	// We expect only LastUpdated to have changed.
	foundNode.LastUpdated = newNode.GetLastUpdated()
	protoassert.Equal(s.T(), foundNode, newNode)
}

func (s *NodesStoreSuite) TestStore_OrphanedCVEs() {
	s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "true")
	if !env.OrphanedCVEsKeepAlive.BooleanSetting() {
		s.T().Skip("Skip tests when ROX_ORPHANED_CVES_KEEP_ALIVE disabled")
		s.T().SkipNow()
	}
	defer s.T().Setenv(env.OrphanedCVEsKeepAlive.EnvVar(), "false")

	store := CreateTableAndNewStore(s.ctx, s.T(), s.pool, s.gormDB, false)

	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))

	foundNode, exists, err := store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundNode)

	s.NoError(store.Upsert(s.ctx, node))

	foundNode, exists, err = store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	s.NotEmpty(foundNode.GetScan().GetComponents())
	s.NotEmpty(foundNode.GetScan().GetComponents()[0].GetVulnerabilities())

	prevVulns := foundNode.GetScan().GetComponents()[0].GetVulnerabilities()
	vulnNames := set.NewStringSet()
	for _, cve := range prevVulns {
		vulnNames.Add(cve.GetCveBaseInfo().GetCve())
	}

	// Remove all node Vulnerabilities
	node = foundNode.CloneVT()
	node.GetScan().GetComponents()[0].Vulnerabilities = nil
	iTime := time.Now()
	node.Scan.ScanTime = protocompat.ConvertTimeToTimestampOrNil(&iTime)
	s.NoError(store.Upsert(s.ctx, node))

	// Updated node does not contain any CVEs
	newNode, exists, err := store.Get(s.ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	s.NotEmpty(newNode.GetScan().GetComponents())
	s.Empty(newNode.GetScan().GetComponents()[0].GetVulnerabilities())

	// Removed vulns should be marked orphaned in node_cves table
	cveStore := nodeCveStore.CreateTableAndNewStore(s.ctx, s.pool, s.gormDB)
	orphanedCVEs, err := cveStore.GetByQuery(s.ctx, search.NewQueryBuilder().AddBools(search.CVEOrphaned, true).ProtoQuery())
	s.NoError(err)
	s.NotEmpty(orphanedCVEs)
	for _, cve := range orphanedCVEs {
		s.NotNil(cve.OrphanedTime)
		s.True(vulnNames.Contains(cve.GetCveBaseInfo().GetCve()))
	}

	orphanedCveIDToCve := make(map[string]*storage.NodeCVE)
	for _, cve := range orphanedCVEs {
		orphanedCveIDToCve[cve.GetId()] = cve
	}

	// Add back prev removed vulnerabilities
	newNode.GetScan().GetComponents()[0].Vulnerabilities = prevVulns
	iTime = time.Now()
	newNode.Scan.ScanTime = protocompat.ConvertTimeToTimestampOrNil(&iTime)
	s.NoError(store.Upsert(s.ctx, newNode))

	// Vulns are added back to the node
	foundNode, exists, err = store.Get(s.ctx, newNode.GetId())
	s.NoError(err)
	s.True(exists)
	s.NotEmpty(newNode.GetScan().GetComponents())
	s.NotEmpty(newNode.GetScan().GetComponents()[0].GetVulnerabilities())

	// CVEs should no longer be marked orphaned
	nodeCVEs, err := cveStore.GetByQuery(s.ctx, search.NewQueryBuilder().AddExactMatches(search.NodeID, foundNode.GetId()).ProtoQuery())
	s.NoError(err)
	s.NotEmpty(nodeCVEs)
	for _, cve := range nodeCVEs {
		s.False(cve.Orphaned)
		s.Nil(cve.OrphanedTime)
		val, ok := orphanedCveIDToCve[cve.GetId()]
		s.True(ok)
		s.Equal(val.GetCveBaseInfo().GetCreatedAt(), cve.GetCveBaseInfo().GetCreatedAt())
	}

	metadatas, missing, err := store.GetManyNodeMetadata(s.ctx, []string{newNode.GetId(), uuid.NewDummy().String()})
	s.NoError(err)
	s.Equal(missing, []int{1})
	protoassert.SlicesEqual(s.T(), []*storage.Node{stripComponents(newNode)}, metadatas)
}

func stripComponents(n *storage.Node) *storage.Node {
	node := n.CloneVT()
	node.GetScan().Components = nil
	return node
}
