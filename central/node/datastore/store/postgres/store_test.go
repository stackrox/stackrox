//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
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
