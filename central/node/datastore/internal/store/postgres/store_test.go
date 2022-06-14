//go:build sql_integration
// +build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/postgres/pgtest"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stackrox/stackrox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

type NodesStoreSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func TestNodesStore(t *testing.T) {
	suite.Run(t, new(NodesStoreSuite))
}

func (s *NodesStoreSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}
}

func (s *NodesStoreSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *NodesStoreSuite) TestStore() {
	ctx := sac.WithAllAccess(context.Background())

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(ctx, config)
	s.NoError(err)
	defer pool.Close()

	Destroy(ctx, pool)

	gormDB := pgtest.OpenGormDB(s.T(), source)
	defer pgtest.CloseGormDB(s.T(), gormDB)
	store := CreateTableAndNewStore(ctx, s.T(), pool, gormDB, false)

	node := &storage.Node{}
	s.NoError(testutils.FullInit(node, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	for _, comp := range node.GetScan().GetComponents() {
		comp.Vulns = nil
	}

	foundNode, exists, err := store.Get(ctx, node.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundNode)

	s.NoError(store.Upsert(ctx, node))
	foundNode, exists, err = store.Get(ctx, node.GetId())
	s.NoError(err)
	s.True(exists)
	cloned := node.Clone()

	for _, component := range cloned.GetScan().GetComponents() {
		for _, vuln := range component.GetVulnerabilities() {
			vuln.CveBaseInfo.CreatedAt = node.GetLastUpdated()
		}
	}
	s.Equal(cloned, foundNode)

	nodeCount, err := store.Count(ctx)
	s.NoError(err)
	s.Equal(nodeCount, 1)

	nodeExists, err := store.Exists(ctx, node.GetId())
	s.NoError(err)
	s.True(nodeExists)
	s.NoError(store.Upsert(ctx, node))

	foundNode, exists, err = store.Get(ctx, node.GetId())
	s.NoError(err)
	s.True(exists)

	// Reconcile the timestamps that are set during upsert.
	cloned.LastUpdated = foundNode.LastUpdated
	s.Equal(cloned, foundNode)

	s.NoError(store.Delete(ctx, node.GetId()))
	foundNode, exists, err = store.Get(ctx, node.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundNode)
}
